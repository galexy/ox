package daemon

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	t.Run("with nil config uses defaults", func(t *testing.T) {
		d := New(nil, nil)
		assert.NotNil(t, d)
		assert.NotNil(t, d.config)
		assert.Equal(t, 5*time.Minute, d.config.SyncIntervalRead)
	})

	t.Run("with custom config", func(t *testing.T) {
		cfg := &Config{
			SyncIntervalRead: 10 * time.Minute,
			LedgerPath:       "/custom/path",
		}
		d := New(cfg, nil)
		assert.Equal(t, 10*time.Minute, d.config.SyncIntervalRead)
		assert.Equal(t, "/custom/path", d.config.LedgerPath)
	})

	t.Run("with custom logger", func(t *testing.T) {
		logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
		d := New(nil, logger)
		assert.Equal(t, logger, d.logger)
	})
}

func TestIsRunning_NoLockFile(t *testing.T) {
	// use temp dir so lock file doesn't exist
	tmpDir := t.TempDir()
	t.Setenv("XDG_RUNTIME_DIR", tmpDir)

	// ensure no lock file
	os.Remove(LockPath())

	assert.False(t, IsRunning())
}

func TestIsRunning_LockFileExists_NotLocked(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_RUNTIME_DIR", tmpDir)

	// create lock file but don't lock it
	lockPath := LockPath()
	err := os.MkdirAll(filepath.Dir(lockPath), 0755)
	require.NoError(t, err)
	f, err := os.Create(lockPath)
	require.NoError(t, err)
	f.Close()

	assert.False(t, IsRunning())
}

func TestDaemon_AcquireLock_Success(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_RUNTIME_DIR", tmpDir)

	d := New(nil, nil)
	err := d.acquireLock()
	require.NoError(t, err)
	assert.NotNil(t, d.lockFile)

	// cleanup
	d.releaseLock()
}

func TestDaemon_AcquireLock_AlreadyRunning(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_RUNTIME_DIR", tmpDir)

	// disable retries so test is fast
	old := lockRetryAttempts
	lockRetryAttempts = 0
	t.Cleanup(func() { lockRetryAttempts = old })

	// first daemon acquires lock
	d1 := New(nil, nil)
	err := d1.acquireLock()
	require.NoError(t, err)

	// second daemon should fail
	d2 := New(nil, nil)
	err = d2.acquireLock()
	assert.ErrorIs(t, err, ErrAlreadyRunning)

	// cleanup
	d1.releaseLock()
}

func TestDaemon_ReleaseLock(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_RUNTIME_DIR", tmpDir)

	d := New(nil, nil)
	err := d.acquireLock()
	require.NoError(t, err)

	d.releaseLock()
	assert.Nil(t, d.lockFile)

	// lock file should be removed
	_, err = os.Stat(LockPath())
	assert.True(t, os.IsNotExist(err))
}

func TestDaemon_ReleaseLock_NilFile(t *testing.T) {
	d := New(nil, nil)
	// should not panic
	d.releaseLock()
}

func TestDaemon_WritePidFile(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_RUNTIME_DIR", tmpDir)

	d := New(nil, nil)
	err := d.writePidFile()
	require.NoError(t, err)

	// verify file exists and contains PID
	content, err := os.ReadFile(PidPath())
	require.NoError(t, err)
	assert.NotEmpty(t, content)
}

func TestDaemon_Cleanup(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_RUNTIME_DIR", tmpDir)

	d := New(nil, nil)
	err := d.acquireLock()
	require.NoError(t, err)

	err = d.writePidFile()
	require.NoError(t, err)

	// create fake socket file
	socketPath := SocketPath()
	f, _ := os.Create(socketPath)
	f.Close()

	d.cleanup()

	// all files should be removed
	_, err = os.Stat(PidPath())
	assert.True(t, os.IsNotExist(err))

	_, err = os.Stat(socketPath)
	assert.True(t, os.IsNotExist(err))

	_, err = os.Stat(LockPath())
	assert.True(t, os.IsNotExist(err))
}

func TestDaemon_Stop_NotRunning(t *testing.T) {
	d := New(nil, nil)
	err := d.Stop()
	assert.ErrorIs(t, err, ErrNotRunning)
}

func TestDaemon_Start_AlreadyRunning(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_RUNTIME_DIR", tmpDir)

	d := New(nil, nil)
	d.running = true
	d.lockFile = &os.File{} // fake lock

	err := d.Start()
	assert.ErrorIs(t, err, ErrAlreadyRunning)
}

// TestLockFileLivenessDetection tests that file locks correctly detect
// daemon liveness even if PID file is stale
func TestLockFileLivenessDetection(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_RUNTIME_DIR", tmpDir)

	// simulate stale PID file (daemon crashed)
	pidPath := PidPath()
	err := os.MkdirAll(filepath.Dir(pidPath), 0755)
	require.NoError(t, err)
	os.WriteFile(pidPath, []byte("99999"), 0644) // fake PID

	// lock is not held, so IsRunning should return false
	assert.False(t, IsRunning(), "should detect daemon is not running despite stale PID")
}

// TestConcurrentLockAcquisition tests that only one daemon can acquire the lock
func TestConcurrentLockAcquisition(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_RUNTIME_DIR", tmpDir)

	// disable retries so concurrent test completes fast
	old := lockRetryAttempts
	lockRetryAttempts = 0
	t.Cleanup(func() { lockRetryAttempts = old })

	const numDaemons = 10
	results := make(chan error, numDaemons)
	daemons := make([]*Daemon, numDaemons)

	// try to acquire lock from multiple goroutines
	for i := 0; i < numDaemons; i++ {
		daemons[i] = New(nil, nil)
		go func(d *Daemon) {
			results <- d.acquireLock()
		}(daemons[i])
	}

	// collect results
	successCount := 0
	for i := 0; i < numDaemons; i++ {
		err := <-results
		if err == nil {
			successCount++
		} else {
			assert.ErrorIs(t, err, ErrAlreadyRunning)
		}
	}

	// exactly one should succeed
	assert.Equal(t, 1, successCount, "exactly one daemon should acquire the lock")

	// cleanup
	for _, d := range daemons {
		if d.lockFile != nil {
			d.releaseLock()
		}
	}
}

// TestAcquireLock_RetrySucceedsAfterRelease tests the restart handoff scenario:
// old daemon holds the lock, new daemon retries, old daemon releases, new daemon acquires.
func TestAcquireLock_RetrySucceedsAfterRelease(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_RUNTIME_DIR", tmpDir)

	// fast retries for test (10 x 10ms = 100ms max wait)
	oldAttempts := lockRetryAttempts
	oldInterval := lockRetryInterval
	lockRetryAttempts = 10
	lockRetryInterval = 10 * time.Millisecond
	t.Cleanup(func() {
		lockRetryAttempts = oldAttempts
		lockRetryInterval = oldInterval
	})

	// old daemon holds the lock
	oldDaemon := New(nil, nil)
	err := oldDaemon.acquireLock()
	require.NoError(t, err)

	// new daemon tries to acquire in background
	newDaemon := New(nil, nil)
	result := make(chan error, 1)
	go func() {
		result <- newDaemon.acquireLock()
	}()

	// release old lock after a brief delay (simulates graceful shutdown completing)
	time.Sleep(30 * time.Millisecond)
	oldDaemon.releaseLock()

	// new daemon should succeed via retry
	err = <-result
	assert.NoError(t, err, "new daemon should acquire lock after old daemon releases")
	assert.NotNil(t, newDaemon.lockFile)

	// cleanup
	newDaemon.releaseLock()
}

// TestAcquireLock_RetryExhaustedStillFails verifies that if the lock is permanently
// held, acquireLock returns ErrAlreadyRunning after exhausting retries.
func TestAcquireLock_RetryExhaustedStillFails(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_RUNTIME_DIR", tmpDir)

	// minimal retries for fast test
	oldAttempts := lockRetryAttempts
	oldInterval := lockRetryInterval
	lockRetryAttempts = 3
	lockRetryInterval = 5 * time.Millisecond
	t.Cleanup(func() {
		lockRetryAttempts = oldAttempts
		lockRetryInterval = oldInterval
	})

	// hold the lock permanently
	holder := New(nil, nil)
	err := holder.acquireLock()
	require.NoError(t, err)
	defer holder.releaseLock()

	// contender should fail after retries
	contender := New(nil, nil)
	start := time.Now()
	err = contender.acquireLock()
	elapsed := time.Since(start)

	assert.ErrorIs(t, err, ErrAlreadyRunning)
	// should have taken at least retries * interval (3 * 5ms = 15ms)
	assert.GreaterOrEqual(t, elapsed.Milliseconds(), int64(15),
		"should have retried before giving up")
}

func TestDaemon_ActivityTracking(t *testing.T) {
	t.Run("initial activity timestamp is set", func(t *testing.T) {
		d := New(nil, nil)
		assert.False(t, d.lastActivity.IsZero())
		// should be recent
		assert.WithinDuration(t, time.Now(), d.lastActivity, time.Second)
	})

	t.Run("recordActivity updates timestamp", func(t *testing.T) {
		d := New(nil, nil)
		initial := d.lastActivity

		// wait a bit
		time.Sleep(10 * time.Millisecond)
		d.recordActivity()

		assert.True(t, d.lastActivity.After(initial))
	})

	t.Run("timeSinceLastActivity returns correct duration", func(t *testing.T) {
		d := New(nil, nil)
		d.lastActivity = time.Now().Add(-5 * time.Minute)

		since := d.timeSinceLastActivity()
		assert.True(t, since >= 5*time.Minute)
		assert.True(t, since < 6*time.Minute)
	})
}

func TestDefaultConfig_NewFields(t *testing.T) {
	cfg := DefaultConfig()

	t.Run("inactivity timeout is set", func(t *testing.T) {
		assert.Equal(t, 1*time.Hour, cfg.InactivityTimeout)
	})

	t.Run("team context sync interval is set", func(t *testing.T) {
		assert.Equal(t, 30*time.Minute, cfg.TeamContextSyncInterval)
	})
}

// TestDaemon_Stop_SetsRunningFalseBeforeCancel verifies that Stop() sets
// running=false before calling cancel(). This ordering is critical to prevent
// goroutines from seeing running=true after context is canceled, which can
// cause use-after-free type bugs where code continues operating on canceled
// resources.
func TestDaemon_Stop_SetsRunningFalseBeforeCancel(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_RUNTIME_DIR", tmpDir)

	d := New(nil, nil)

	// simulate daemon in running state
	d.running = true
	ctx, cancel := context.WithCancel(context.Background())
	d.cancel = cancel
	d.lockFile = &os.File{} // fake lock to satisfy cleanup

	// channel to communicate result from observer goroutine
	resultCh := make(chan bool, 1)

	// observer goroutine that checks running state when context is canceled
	go func() {
		<-ctx.Done()
		d.mu.Lock()
		wasRunning := d.running
		d.mu.Unlock()
		resultCh <- wasRunning
	}()

	// call Stop
	err := d.Stop()
	require.NoError(t, err)

	// wait for observer to report
	select {
	case wasRunning := <-resultCh:
		assert.False(t, wasRunning,
			"running should be false when context is canceled to prevent race conditions")
	case <-time.After(time.Second):
		t.Fatal("observer goroutine did not complete")
	}
}

// TestDaemon_ConcurrentStop_NoRace tests that concurrent Stop() calls don't
// cause race conditions. Run with -race flag to detect issues.
func TestDaemon_ConcurrentStop_NoRace(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_RUNTIME_DIR", tmpDir)

	d := New(nil, nil)
	d.running = true
	_, cancel := context.WithCancel(context.Background())
	d.cancel = cancel
	d.lockFile = &os.File{}

	// call Stop concurrently
	const numGoroutines = 10
	done := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			done <- d.Stop()
		}()
	}

	// collect results - only one should succeed, rest should get ErrNotRunning
	successCount := 0
	notRunningCount := 0
	for i := 0; i < numGoroutines; i++ {
		err := <-done
		switch err {
		case nil:
			successCount++
		case ErrNotRunning:
			notRunningCount++
		}
	}

	// exactly one should succeed
	assert.Equal(t, 1, successCount, "exactly one Stop should succeed")
	assert.Equal(t, numGoroutines-1, notRunningCount, "others should get ErrNotRunning")
}
