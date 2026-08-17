package server

import (
	"bufio"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/benedictjohannes/crobe/playbook"
)

func TestEventBroker(t *testing.T) {
	broker := NewEventBroker()

	if broker.SubscriberCount() != 0 {
		t.Fatalf("expected 0 subscribers, got %d", broker.SubscriberCount())
	}

	ch1 := broker.Subscribe()
	ch2 := broker.Subscribe()

	if broker.SubscriberCount() != 2 {
		t.Fatalf("expected 2 subscribers, got %d", broker.SubscriberCount())
	}

	id1 := broker.Broadcast("state_change", "run-1", map[string]string{"status": "running"})
	if id1 != 1 {
		t.Errorf("expected event ID 1, got %d", id1)
	}

	id2 := broker.Broadcast("log", "run-1", LogEventData{RunID: "run-1", Message: "hello"})
	if id2 != 2 {
		t.Errorf("expected event ID 2, got %d", id2)
	}

	// Verify ch1 received both events
	select {
	case ev := <-ch1:
		if ev.ID != 1 || ev.Type != "state_change" {
			t.Errorf("ch1 received unexpected ev1: %+v", ev)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("timed out waiting for ev1 on ch1")
	}

	select {
	case ev := <-ch1:
		if ev.ID != 2 || ev.Type != "log" {
			t.Errorf("ch1 received unexpected ev2: %+v", ev)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("timed out waiting for ev2 on ch1")
	}

	// Verify ch2 received both events
	select {
	case ev := <-ch2:
		if ev.ID != 1 {
			t.Errorf("ch2 received unexpected ev1 ID: %d", ev.ID)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("timed out waiting for ev1 on ch2")
	}

	select {
	case ev := <-ch2:
		if ev.ID != 2 {
			t.Errorf("ch2 received unexpected ev2 ID: %d", ev.ID)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("timed out waiting for ev2 on ch2")
	}

	broker.Unsubscribe(ch1)
	if broker.SubscriberCount() != 1 {
		t.Fatalf("expected 1 subscriber after unsubscribe, got %d", broker.SubscriberCount())
	}

	broker.Unsubscribe(ch2)
	if broker.SubscriberCount() != 0 {
		t.Fatalf("expected 0 subscribers after unsubscribe, got %d", broker.SubscriberCount())
	}
}

func TestSSEEndpointStream(t *testing.T) {
	srv, err := NewServer(Config{
		Token:               "testtoken123456789012345678901234",
		DisableAutoShutdown: true,
	})
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}
	defer srv.Close()

	ts := httptest.NewServer(srv.httpServer.Handler)
	defer ts.Close()

	// 1. Unauthenticated request should return 401
	reqUnauth, _ := http.NewRequest(http.MethodGet, ts.URL+"/api/events", nil)
	respUnauth, err := http.DefaultClient.Do(reqUnauth)
	if err != nil {
		t.Fatalf("failed to make unauthenticated request: %v", err)
	}
	if respUnauth.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401 for unauthenticated /api/events, got %d", respUnauth.StatusCode)
	}
	respUnauth.Body.Close()

	// 2. Authenticated SSE Stream
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, ts.URL+"/api/events", nil)
	req.Header.Set("Authorization", "Bearer testtoken123456789012345678901234")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("failed to connect to /api/events: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK for /api/events, got %d", resp.StatusCode)
	}
	if contentType := resp.Header.Get("Content-Type"); !strings.HasPrefix(contentType, "text/event-stream") {
		t.Errorf("expected Content-Type text/event-stream, got %s", contentType)
	}

	reader := bufio.NewReader(resp.Body)

	// Read initial connected comment
	initialLine, err := reader.ReadString('\n')
	if err != nil {
		t.Fatalf("failed to read initial SSE line: %v", err)
	}
	if !strings.HasPrefix(initialLine, ": connected") {
		t.Errorf("expected ': connected', got %q", initialLine)
	}
	// Consume blank line
	_, _ = reader.ReadString('\n')

	// 3. Broadcast events and verify parsing on client
	go func() {
		time.Sleep(50 * time.Millisecond)
		srv.EventBroker().Broadcast("state_change", "run-100", map[string]string{"status": "loaded"})
		srv.EventBroker().Broadcast("log", "run-100", LogEventData{RunID: "run-100", Message: "Test Log 1"})
		srv.EventBroker().Broadcast("assertion_progress", "run-100", AssertionProgressEventData{
			RunID:    "run-100",
			Code:     "A1",
			Status:   "passed",
			Passed:   true,
			Score:    100,
			MinScore: 100,
		})
		srv.EventBroker().Broadcast("execution_completed", "run-100", ExecutionCompletedEventData{
			RunID:      "run-100",
			Status:     "completed",
			DurationMs: 250,
		})
	}()

	// Helper to read a single SSE message
	readSSEMessage := func() (id string, eventType string, data string) {
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				t.Fatalf("failed to read SSE message line: %v", err)
			}
			line = strings.TrimRight(line, "\r\n")
			if line == "" {
				if eventType != "" {
					return
				}
				continue
			}
			if strings.HasPrefix(line, "id: ") {
				id = strings.TrimPrefix(line, "id: ")
			} else if strings.HasPrefix(line, "event: ") {
				eventType = strings.TrimPrefix(line, "event: ")
			} else if strings.HasPrefix(line, "data: ") {
				data = strings.TrimPrefix(line, "data: ")
			}
		}
	}

	// 1st message: state_change
	id1, ev1, data1 := readSSEMessage()
	if id1 != "1" || ev1 != "state_change" || !strings.Contains(data1, "loaded") {
		t.Errorf("unexpected event 1: id=%s ev=%s data=%s", id1, ev1, data1)
	}

	// 2nd message: log
	id2, ev2, data2 := readSSEMessage()
	if id2 != "2" || ev2 != "log" || !strings.Contains(data2, "Test Log 1") {
		t.Errorf("unexpected event 2: id=%s ev=%s data=%s", id2, ev2, data2)
	}

	// 3rd message: assertion_progress
	id3, ev3, data3 := readSSEMessage()
	if id3 != "3" || ev3 != "assertion_progress" || !strings.Contains(data3, "A1") {
		t.Errorf("unexpected event 3: id=%s ev=%s data=%s", id3, ev3, data3)
	}

	// 4th message: execution_completed
	id4, ev4, data4 := readSSEMessage()
	if id4 != "4" || ev4 != "execution_completed" || !strings.Contains(data4, "completed") {
		t.Errorf("unexpected event 4: id=%s ev=%s data=%s", id4, ev4, data4)
	}
}

func TestSSEMultiClientBroadcasting(t *testing.T) {
	srv, err := NewServer(Config{
		Token:               "multiclienttesttoken123456789012",
		DisableAutoShutdown: true,
	})
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}
	defer srv.Close()

	ts := httptest.NewServer(srv.httpServer.Handler)
	defer ts.Close()

	const numClients = 3
	var wg sync.WaitGroup
	wg.Add(numClients)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	for i := 0; i < numClients; i++ {
		go func(clientIdx int) {
			defer wg.Done()
			req, _ := http.NewRequestWithContext(ctx, http.MethodGet, ts.URL+"/api/events", nil)
			req.Header.Set("Authorization", "Bearer multiclienttesttoken123456789012")

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Errorf("client %d failed to connect: %v", clientIdx, err)
				return
			}
			defer resp.Body.Close()

			reader := bufio.NewReader(resp.Body)
			// Read connected header
			_, _ = reader.ReadString('\n')
			_, _ = reader.ReadString('\n')

			// Read broadcasted event
			for {
				line, err := reader.ReadString('\n')
				if err != nil {
					return
				}
				if strings.HasPrefix(line, "event: ") {
					evType := strings.TrimSpace(strings.TrimPrefix(line, "event: "))
					if evType == "log" {
						return // Success
					}
				}
			}
		}(i)
	}

	// Wait for all clients to subscribe
	time.Sleep(100 * time.Millisecond)

	srv.EventBroker().Broadcast("log", "run-multi", LogEventData{
		RunID:   "run-multi",
		Message: "Multi-client broadcast verification",
	})

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// All clients received the event
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for all multi-client subscribers to receive event")
	}
}

func TestSSEExecutionLiveEvents(t *testing.T) {
	srv, err := NewServer(Config{
		Token:               "liveexecutionevents123456789012",
		DisableAutoShutdown: true,
	})
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}
	defer srv.Close()

	pb := &playbook.Playbook{
		Title: "Live Test",
		Sections: []playbook.Section{
			{
				Title: "S1",
				Assertions: []playbook.Assertion{
					{
						Code:  "L1",
						Title: "L1",
						Cmds: []playbook.Cmd{
							{
								Exec: playbook.Exec{
									Script: "echo live-test-output",
								},
							},
						},
					},
				},
			},
		},
	}
	srv.StateManager().SetPlaybook(pb, []byte("title: Live Test"), nil)

	ts := httptest.NewServer(srv.httpServer.Handler)
	defer ts.Close()

	clientEvents := &http.Client{Transport: &http.Transport{}}
	clientRun := &http.Client{Transport: &http.Transport{}}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, ts.URL+"/api/events", nil)
	req.Header.Set("Authorization", "Bearer liveexecutionevents123456789012")

	resp, err := clientEvents.Do(req)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer resp.Body.Close()

	reader := bufio.NewReader(resp.Body)

	// Read events asynchronously with channel and timeout
	eventsReceived := make(map[string]bool)
	lineChan := make(chan string, 200)
	errChan := make(chan error, 1)

	go func() {
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				errChan <- err
				return
			}
			lineChan <- line
		}
	}()

	// Read connected header lines
	select {
	case line := <-lineChan:
		if !strings.HasPrefix(line, ": connected") {
			t.Errorf("expected ': connected', got %q", line)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for connected SSE header")
	}

	// Trigger run
	runReq, _ := http.NewRequest(http.MethodPost, ts.URL+"/api/run", nil)
	runReq.Header.Set("Authorization", "Bearer liveexecutionevents123456789012")
	runResp, err := clientRun.Do(runReq)
	if err != nil {
		t.Fatalf("failed to trigger run: %v", err)
	}
	if runResp.StatusCode != http.StatusAccepted {
		t.Fatalf("expected 202 Accepted, got %d", runResp.StatusCode)
	}
	runResp.Body.Close()

	timeout := time.After(5 * time.Second)

	for {
		select {
		case <-timeout:
			t.Fatalf("timed out waiting for events, received so far: %v", eventsReceived)
		case err := <-errChan:
			t.Fatalf("error reading SSE stream: %v", err)
		case line := <-lineChan:
			t.Logf("SSE line received: %s", strings.TrimSpace(line))
			if strings.HasPrefix(line, "event: ") {
				ev := strings.TrimSpace(strings.TrimPrefix(line, "event: "))
				eventsReceived[ev] = true
			}
			if eventsReceived["execution_completed"] && eventsReceived["assertion_progress"] && eventsReceived["state_change"] {
				return // Success
			}
		}
	}
}
