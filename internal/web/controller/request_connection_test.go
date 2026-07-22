package controller

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

type deadlineRecordingConn struct {
	readDeadline  time.Time
	writeDeadline time.Time
}

func (*deadlineRecordingConn) Read([]byte) (int, error)    { return 0, net.ErrClosed }
func (*deadlineRecordingConn) Write([]byte) (int, error)   { return 0, net.ErrClosed }
func (*deadlineRecordingConn) Close() error                { return nil }
func (*deadlineRecordingConn) LocalAddr() net.Addr         { return nil }
func (*deadlineRecordingConn) RemoteAddr() net.Addr        { return nil }
func (*deadlineRecordingConn) SetDeadline(time.Time) error { return nil }
func (c *deadlineRecordingConn) SetReadDeadline(value time.Time) error {
	c.readDeadline = value
	return nil
}

func (c *deadlineRecordingConn) SetWriteDeadline(value time.Time) error {
	c.writeDeadline = value
	return nil
}

func TestExtendRequestConnectionDeadlineIsRouteScoped(t *testing.T) {
	connection := &deadlineRecordingConn{}
	ctx := WithRequestConnection(context.Background(), connection)
	before := time.Now().Add(90 * time.Minute)
	if !extendRequestConnectionDeadline(ctx, 2*time.Hour) {
		t.Fatal("request connection deadline was not extended")
	}
	if connection.readDeadline.Before(before) || connection.writeDeadline.Before(before) {
		t.Fatalf("streaming deadlines were not extended: read=%v write=%v", connection.readDeadline, connection.writeDeadline)
	}
	if extendRequestConnectionDeadline(context.Background(), 2*time.Hour) {
		t.Fatal("context without a server connection unexpectedly reported success")
	}
}

type slowRequestBody struct {
	remaining int
	delay     time.Duration
}

func (body *slowRequestBody) Read(buffer []byte) (int, error) {
	if body.remaining == 0 {
		return 0, io.EOF
	}
	time.Sleep(body.delay)
	buffer[0] = 'x'
	body.remaining--
	return 1, nil
}

func TestStreamingRouteOverridesDefaultHTTPServerDeadline(t *testing.T) {
	handler := http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if !extendRequestConnectionDeadline(request.Context(), 2*time.Second) {
			http.Error(writer, "connection unavailable", http.StatusInternalServerError)
			return
		}
		if _, err := io.Copy(io.Discard, request.Body); err != nil {
			http.Error(writer, err.Error(), http.StatusRequestTimeout)
			return
		}
		writer.WriteHeader(http.StatusNoContent)
	})
	server := httptest.NewUnstartedServer(handler)
	server.Config.ConnContext = WithRequestConnection
	server.Config.ReadTimeout = 50 * time.Millisecond
	server.Config.WriteTimeout = 50 * time.Millisecond
	server.Start()
	defer server.Close()

	response, err := server.Client().Post(server.URL, "application/octet-stream", &slowRequestBody{remaining: 4, delay: 80 * time.Millisecond})
	if err != nil {
		t.Fatalf("slow streaming request failed despite route override: %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusNoContent {
		t.Fatalf("slow streaming status = %d, want %d", response.StatusCode, http.StatusNoContent)
	}
}
