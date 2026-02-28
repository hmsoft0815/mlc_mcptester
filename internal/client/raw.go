package client

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"unsafe"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// CallToolRaw performs a tool call and returns the raw map[string]any result,
// bypassing the strict SDK unmarshaling that fails on missing "type" fields.
func CallToolRaw(ctx context.Context, session *mcp.ClientSession, toolName string, arguments any) (map[string]any, error) {
	// 1. Get the internal jsonrpc2.Connection via reflection on unexported field
	sVal := reflect.ValueOf(session).Elem()
	connField := sVal.FieldByName("conn")
	if !connField.IsValid() {
		return nil, fmt.Errorf("could not find 'conn' field in ClientSession")
	}

	// Create a NEW reflect.Value that is addressable and exported (using NewAt)
	connPtr := reflect.NewAt(connField.Type().Elem(), unsafe.Pointer(connField.Pointer()))
	
	// 2. Prepare the request
	params := struct {
		Name      string `json:"name"`
		Arguments any    `json:"arguments"`
	}{
		Name:      toolName,
		Arguments: arguments,
	}
	
	// 3. Call the internal Connection.Call method
	callMethod := connPtr.MethodByName("Call")
	if !callMethod.IsValid() {
		return nil, fmt.Errorf("Call method not found on Connection")
	}
	
	callResults := callMethod.Call([]reflect.Value{
		reflect.ValueOf(ctx),
		reflect.ValueOf("tools/call"),
		reflect.ValueOf(params),
	})
	
	asyncCall := callResults[0]
	
	// 4. Wait for ready
	// asyncCall is *AsyncCall. We need to access its unexported field 'ready'
	// Use NewAt to get an exported version of the channel field
	acType := asyncCall.Type().Elem()
	readyField, _ := acType.FieldByName("ready")
	readyPtr := reflect.NewAt(readyField.Type, unsafe.Pointer(uintptr(unsafe.Pointer(asyncCall.Pointer()))+readyField.Offset)).Elem()

	done := make(chan struct{})
	go func() {
		readyPtr.Recv()
		close(done)
	}()

	select {
	case <-done:
		// Call is ready
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	
	// 5. Get response
	respFieldInfo, _ := acType.FieldByName("response")
	respPtr := reflect.NewAt(respFieldInfo.Type, unsafe.Pointer(uintptr(unsafe.Pointer(asyncCall.Pointer()))+respFieldInfo.Offset)).Elem()
	
	if respPtr.IsNil() {
		return nil, fmt.Errorf("response is nil after call ready")
	}
	
	// respPtr is *Response (internal/jsonrpc2.Response)
	respVal := reflect.NewAt(respPtr.Type().Elem(), unsafe.Pointer(respPtr.Pointer())).Elem()
	
	// Error check
	errField := respVal.FieldByName("Error")
	if !errField.IsNil() {
		return nil, errField.Interface().(error)
	}
	
	// Result check
	resultField := respVal.FieldByName("Result")
	// resultField is json.RawMessage ([]byte)
	rawJSON := resultField.Bytes()
	
	var resultMap map[string]any
	if err := json.Unmarshal(rawJSON, &resultMap); err != nil {
		return nil, fmt.Errorf("failed to unmarshal raw response: %w", err)
	}
	
	return resultMap, nil
}
