package config

import (
	"sync"
	"testing"
	"time"
)

func TestDeadlockReproAddGetDelete(t *testing.T) {
	useTestConfigDir(t)
	done := make(chan struct{})
	go func() {
		defer close(done)
		if err := AddAccount("u1", "p1"); err != nil {
			t.Error(err)
			return
		}
		if _, err := GetAccount("u1"); err != nil {
			t.Error(err)
			return
		}
		if err := AddAccount("u1", "p2"); err != nil {
			t.Error(err)
			return
		}
		if err := DeleteAccount("u1"); err != nil {
			t.Error(err)
			return
		}
		var wg sync.WaitGroup
		for i := 0; i < 5; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_ = AddAccount("concurrent", "p")
			}()
		}
		wg.Wait()
	}()
	select {
	case <-done:
	case <-time.After(20 * time.Second):
		t.Fatal("deadlock: account ops did not complete")
	}
}
