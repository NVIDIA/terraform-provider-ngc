package utils

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-log/tfsdklog"
)

// sendRequest logs request_body and response_body at DEBUG. When a cloud
// function declares secrets, NvidiaCloudFunctionSecret.Value carries the
// plaintext, so both fields must be masked before the log is emitted.
//
// These tests exercise the masking mechanism through the real tfsdklog sink,
// mirroring the exact call sequence in sendRequest.
const logCanary = "s3cr3t-canary-value-do-not-log"

// captureTFLog wires a real provider logger to a file and returns its contents.
func captureTFLog(t *testing.T, emit func(ctx context.Context)) string {
	t.Helper()
	logPath := filepath.Join(t.TempDir(), "tf.log")
	t.Setenv("TF_LOG", "TRACE")
	t.Setenv("TF_LOG_PATH", logPath)

	ctx := tfsdklog.RegisterTestSink(context.Background(), t)
	ctx = tfsdklog.NewRootProviderLogger(ctx)
	emit(ctx)

	b, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("reading captured log: %v", err)
	}
	return string(b)
}

func requestWithSecret() *CreateNvidiaCloudFunctionRequest {
	return &CreateNvidiaCloudFunctionRequest{
		Secrets: []NvidiaCloudFunctionSecret{{Name: "api_key", Value: logCanary}},
	}
}

func TestSecretsAreMaskedInDebugLog(t *testing.T) {
	out := captureTFLog(t, func(ctx context.Context) {
		// Same order as sendRequest.
		ctx = tflog.MaskFieldValuesWithFieldKeys(ctx, "request_body", "response_body")
		ctx = tflog.SetField(ctx, "request_method", "POST")
		ctx = tflog.SetField(ctx, "response_body", `{"secrets":[{"value":"`+logCanary+`"}]}`)
		ctx = tflog.SetField(ctx, "request_body", requestWithSecret())
		tflog.Debug(ctx, "Send request")
	})

	if strings.Contains(out, logCanary) {
		t.Fatalf("secret leaked into debug log:\n%s", out)
	}
	if !strings.Contains(out, "request_method") {
		t.Fatalf("masking removed non-sensitive fields too:\n%s", out)
	}
}

// Control: the identical sequence WITHOUT the mask must leak. If this stops
// failing to leak, the test above has stopped proving anything.
func TestControlUnmaskedSecretDoesLeak(t *testing.T) {
	out := captureTFLog(t, func(ctx context.Context) {
		ctx = tflog.SetField(ctx, "request_body", requestWithSecret())
		tflog.Debug(ctx, "Send request")
	})

	if !strings.Contains(out, logCanary) {
		t.Fatalf("control failed: expected the unmasked secret to appear, got:\n%s", out)
	}
}

// The encode-failure path logs at ERROR, which is emitted at every TF_LOG
// level, so it must not format the request body into the message.
func TestEncodeFailureMessageOmitsRequestBody(t *testing.T) {
	out := captureTFLog(t, func(ctx context.Context) {
		tflog.Error(ctx, "failed to encode request body")
	})
	if strings.Contains(out, logCanary) {
		t.Fatalf("secret leaked via the error path:\n%s", out)
	}
}

// The unexpected-status path returns a Go error. A returned error is NOT the
// tflog sink: Terraform renders it via resp.Diagnostics.AddError(..., err.Error())
// straight to the CLI, so tflog.MaskFieldValuesWithFieldKeys cannot redact it.
// Any response body embedded in that error therefore reaches the console
// unmasked. Assert the error carries only the status, never the body.
func TestErrorMessageOmitsResponseBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		// Not valid JSON, so ErrorResponse unmarshalling fails and the
		// "failed to parse error response body" branch is taken.
		_, _ = w.Write([]byte("<html>upstream proxy error: token=" + logCanary + "</html>"))
	}))
	defer server.Close()

	c := &NVCFClient{
		NgcEndpoint: server.URL,
		NgcApiKey:   "MOCK_API_KEY",
		NgcOrg:      "MOCK_ORG",
		HttpClient:  server.Client(),
	}

	err := c.sendRequest(
		context.Background(),
		server.URL,
		http.MethodPost,
		requestWithSecret(),
		nil,
		map[int]bool{http.StatusOK: true},
		nil,
	)
	if err == nil {
		t.Fatal("expected an error for the unexpected status code")
	}
	if strings.Contains(err.Error(), logCanary) {
		t.Fatalf("response body leaked through the returned error: %s", err.Error())
	}
	if !strings.Contains(err.Error(), "502") {
		t.Fatalf("error should still identify the status code, got: %s", err.Error())
	}
}
