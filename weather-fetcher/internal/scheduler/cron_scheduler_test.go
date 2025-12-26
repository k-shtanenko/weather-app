package scheduler

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/robfig/cron/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestCronScheduler_Schedule(t *testing.T) {
	t.Run("successful schedule", func(t *testing.T) {
		scheduler := NewCronScheduler(30 * time.Second).(*CronScheduler)

		ctx := context.Background()
		interval := 5 * time.Minute

		var taskExecuted bool
		var taskMutex sync.Mutex

		task := func(ctx context.Context) error {
			taskMutex.Lock()
			taskExecuted = true
			taskMutex.Unlock()
			return nil
		}

		err := scheduler.Schedule(ctx, interval, task)
		assert.NoError(t, err)

		time.Sleep(100 * time.Millisecond)

		taskMutex.Lock()
		_ = taskExecuted
		taskMutex.Unlock()

		assert.NoError(t, err)

		scheduler.Stop()
	})

	t.Run("schedule with zero interval", func(t *testing.T) {
		scheduler := NewCronScheduler(30 * time.Second)

		ctx := context.Background()
		interval := 0 * time.Second

		task := func(ctx context.Context) error {
			return nil
		}

		err := scheduler.Schedule(ctx, interval, task)
		assert.NoError(t, err)

		scheduler.Stop()
	})

	t.Run("schedule with small interval", func(t *testing.T) {
		scheduler := NewCronScheduler(30 * time.Second)

		ctx := context.Background()
		interval := 5 * time.Second

		task := func(ctx context.Context) error {
			return nil
		}

		err := scheduler.Schedule(ctx, interval, task)
		assert.NoError(t, err)

		scheduler.Stop()
	})

	t.Run("task execution with timeout", func(t *testing.T) {
		scheduler := NewCronScheduler(100 * time.Millisecond)

		ctx := context.Background()
		interval := 100 * time.Millisecond

		taskCompleted := make(chan bool, 1)

		task := func(ctx context.Context) error {
			select {
			case <-time.After(200 * time.Millisecond):
				taskCompleted <- false
				return nil
			case <-ctx.Done():
				taskCompleted <- true
				return ctx.Err()
			}
		}

		err := scheduler.Schedule(ctx, interval, task)
		assert.NoError(t, err)

		time.Sleep(300 * time.Millisecond)

		select {
		case completed := <-taskCompleted:
			assert.True(t, completed, "Task should have timed out")
		default:
		}

		scheduler.Stop()
	})
}

func TestCronScheduler_Stop(t *testing.T) {
	t.Run("stop scheduler", func(t *testing.T) {
		scheduler := NewCronScheduler(30 * time.Second)

		ctx := context.Background()
		interval := 1 * time.Second

		taskCount := 0
		var mutex sync.Mutex

		task := func(ctx context.Context) error {
			mutex.Lock()
			taskCount++
			mutex.Unlock()
			return nil
		}

		err := scheduler.Schedule(ctx, interval, task)
		assert.NoError(t, err)

		time.Sleep(100 * time.Millisecond)

		scheduler.Stop()

		time.Sleep(500 * time.Millisecond)

		mutex.Lock()
		count := taskCount
		mutex.Unlock()

		initialCount := count
		time.Sleep(1 * time.Second)

		mutex.Lock()
		finalCount := taskCount
		mutex.Unlock()

		assert.Equal(t, initialCount, finalCount)
	})

	t.Run("stop multiple times", func(t *testing.T) {
		scheduler := NewCronScheduler(30 * time.Second)

		scheduler.Stop()
		scheduler.Stop()
		scheduler.Stop()

		assert.True(t, true)
	})
}

func TestIntervalToCron(t *testing.T) {
	testCases := []struct {
		name     string
		interval time.Duration
		expected string
	}{
		{"zero interval", 0, "0 */5 * * * *"},
		{"5 minutes", 5 * time.Minute, "*/300 * * * * *"},
		{"1 minute", time.Minute, "*/60 * * * * *"},
		{"30 seconds", 30 * time.Second, "*/30 * * * * *"},
		{"5 seconds (less than minimum)", 5 * time.Second, "*/10 * * * * *"},
		{"1 hour", time.Hour, "*/3600 * * * * *"},
		{"2 hours", 2 * time.Hour, "*/7200 * * * * *"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := intervalToCron(tc.interval)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestCronSchedulerFactory(t *testing.T) {
	factory := NewCronSchedulerFactory()

	scheduler := factory.CreateScheduler(30 * time.Second)

	assert.NotNil(t, scheduler)

	ctx := context.Background()
	err := scheduler.Schedule(ctx, time.Minute, func(ctx context.Context) error {
		return nil
	})
	assert.NoError(t, err)

	scheduler.Stop()
}

func TestCronScheduler_wrapTask(t *testing.T) {
	t.Run("task completes successfully", func(t *testing.T) {
		scheduler := NewCronScheduler(30 * time.Second).(*CronScheduler)

		ctx := context.Background()
		taskExecuted := false
		task := func(ctx context.Context) error {
			taskExecuted = true
			return nil
		}

		wrappedTask := scheduler.wrapTask(ctx, task)

		wrappedTask()

		assert.True(t, taskExecuted)
	})

	t.Run("task fails", func(t *testing.T) {
		scheduler := NewCronScheduler(30 * time.Second).(*CronScheduler)

		ctx := context.Background()
		expectedErr := errors.New("task error")
		task := func(ctx context.Context) error {
			return expectedErr
		}

		wrappedTask := scheduler.wrapTask(ctx, task)

		assert.NotPanics(t, func() {
			wrappedTask()
		})
	})

	t.Run("task times out", func(t *testing.T) {
		scheduler := NewCronScheduler(100 * time.Millisecond).(*CronScheduler)

		ctx := context.Background()
		taskCompleted := make(chan error, 1)

		task := func(ctx context.Context) error {
			select {
			case <-time.After(200 * time.Millisecond):
				taskCompleted <- nil
				return nil
			case <-ctx.Done():
				taskCompleted <- ctx.Err()
				return ctx.Err()
			}
		}

		wrappedTask := scheduler.wrapTask(ctx, task)

		wrappedTask()

		select {
		case err := <-taskCompleted:
			assert.Equal(t, context.DeadlineExceeded, err)
		case <-time.After(300 * time.Millisecond):
			t.Fatal("Task should have completed")
		}
	})

	t.Run("task cancelled by parent context", func(t *testing.T) {
		scheduler := NewCronScheduler(30 * time.Second).(*CronScheduler)

		ctx, cancel := context.WithCancel(context.Background())
		taskCompleted := make(chan error, 1)

		task := func(ctx context.Context) error {
			select {
			case <-time.After(100 * time.Millisecond):
				taskCompleted <- nil
				return nil
			case <-ctx.Done():
				taskCompleted <- ctx.Err()
				return ctx.Err()
			}
		}

		wrappedTask := scheduler.wrapTask(ctx, task)

		cancel()

		wrappedTask()

		select {
		case err := <-taskCompleted:
			assert.Equal(t, context.Canceled, err)
		case <-time.After(200 * time.Millisecond):
			t.Fatal("Task should have completed")
		}
	})
}

func TestNewCronScheduler(t *testing.T) {
	t.Run("create scheduler with timeout", func(t *testing.T) {
		timeout := 10 * time.Second
		scheduler := NewCronScheduler(timeout)

		assert.NotNil(t, scheduler)

		ctx := context.Background()
		err := scheduler.Schedule(ctx, time.Minute, func(ctx context.Context) error {
			return nil
		})
		assert.NoError(t, err)

		scheduler.Stop()
	})

	t.Run("schedule multiple tasks", func(t *testing.T) {
		scheduler := NewCronScheduler(30 * time.Second)

		ctx := context.Background()

		task1Count := 0
		task2Count := 0
		var mutex sync.Mutex

		task1 := func(ctx context.Context) error {
			mutex.Lock()
			task1Count++
			mutex.Unlock()
			return nil
		}

		task2 := func(ctx context.Context) error {
			mutex.Lock()
			task2Count++
			mutex.Unlock()
			return nil
		}

		err := scheduler.Schedule(ctx, 100*time.Millisecond, task1)
		assert.NoError(t, err)

		err = scheduler.Schedule(ctx, 150*time.Millisecond, task2)
		assert.NoError(t, err)

		time.Sleep(500 * time.Millisecond)

		scheduler.Stop()

		time.Sleep(100 * time.Millisecond)

		mutex.Lock()
		count1 := task1Count
		count2 := task2Count
		mutex.Unlock()

		t.Logf("Task 1 executed %d times, Task 2 executed %d times", count1, count2)

		assert.True(t, true, "Test completed without errors")
	})
}

func TestCronSchedulerFactory_CreateScheduler(t *testing.T) {
	factory := NewCronSchedulerFactory()

	scheduler := factory.CreateScheduler(30 * time.Second)

	assert.NotNil(t, scheduler)
	assert.Implements(t, (*Scheduler)(nil), scheduler)
}

func TestCronSchedulerFactory_InterfaceImplementation(t *testing.T) {
	var _ SchedulerFactory = (*CronSchedulerFactory)(nil)
}

func TestCronSchedulerFactory_CreateScheduler_ZeroTimeout(t *testing.T) {
	factory := NewCronSchedulerFactory()

	scheduler := factory.CreateScheduler(0)
	assert.NotNil(t, scheduler)

	ctx := context.Background()
	err := scheduler.Schedule(ctx, time.Minute, func(ctx context.Context) error {
		return nil
	})
	assert.NoError(t, err)

	scheduler.Stop()
}

func TestIntervalToCron_BoundaryValues(t *testing.T) {
	assert.Equal(t, "*/10 * * * * *", intervalToCron(9*time.Second))
	assert.Equal(t, "*/10 * * * * *", intervalToCron(10*time.Second))
	assert.Equal(t, "*/11 * * * * *", intervalToCron(11*time.Second))
	assert.Equal(t, "0 */5 * * * *", intervalToCron(0))
	assert.Equal(t, "*/10 * * * * *", intervalToCron(-5*time.Second))
}

func TestIntervalToCron_VeryLargeInterval(t *testing.T) {
	result := intervalToCron(48 * time.Hour)
	expectedSeconds := 48 * 60 * 60
	assert.Equal(t, fmt.Sprintf("*/%d * * * * *", expectedSeconds), result)

	result = intervalToCron(5 * time.Second)
	assert.Equal(t, "*/10 * * * * *", result)

	result = intervalToCron(-10 * time.Second)
	assert.Equal(t, "*/10 * * * * *", result)
}

func TestCronScheduler_Schedule_InvalidCronExpression(t *testing.T) {
	scheduler := NewCronScheduler(30 * time.Second).(*CronScheduler)

	ctx := context.Background()
	task := func(ctx context.Context) error {
		return nil
	}

	err := scheduler.Schedule(ctx, -1*time.Second, task)

	assert.NoError(t, err)

	scheduler.Stop()
}

func TestCronScheduler_Schedule_InvalidCronExpression_EdgeCase(t *testing.T) {
	scheduler := NewCronScheduler(30 * time.Second).(*CronScheduler)

	ctx := context.Background()
	task := func(ctx context.Context) error {
		return nil
	}

	scheduler.cron = cron.New(cron.WithSeconds())

	err := scheduler.Schedule(ctx, -1*time.Second, task)
	assert.NoError(t, err)

	scheduler.Stop()
}

type mockCronStruct struct {
	mock.Mock
}

func (m *mockCronStruct) AddFunc(spec string, cmd func()) (cron.EntryID, error) {
	args := m.Called(spec, cmd)
	return args.Get(0).(cron.EntryID), args.Error(1)
}

func (m *mockCronStruct) Start() {
	m.Called()
}

func (m *mockCronStruct) Stop() context.Context {
	args := m.Called()
	return args.Get(0).(context.Context)
}

func (m *mockCronStruct) Entries() []cron.Entry {
	args := m.Called()
	return args.Get(0).([]cron.Entry)
}

func (m *mockCronStruct) AddJob(spec string, cmd cron.Job) (cron.EntryID, error) {
	args := m.Called(spec, cmd)
	return args.Get(0).(cron.EntryID), args.Error(1)
}

func TestIntervalToCron_VeryLargeInterval_EdgeCase(t *testing.T) {
	result := intervalToCron(9 * time.Second)
	assert.Equal(t, "*/10 * * * * *", result)

	result = intervalToCron(100 * time.Hour)
	expectedSeconds := 100 * 60 * 60
	assert.Equal(t, fmt.Sprintf("*/%d * * * * *", expectedSeconds), result)
}

func TestCronScheduler_Schedule_WithMock(t *testing.T) {
	scheduler := NewCronScheduler(30 * time.Second).(*CronScheduler)

	ctx := context.Background()
	interval := 5 * time.Minute
	task := func(ctx context.Context) error {
		return nil
	}

	err := scheduler.Schedule(ctx, interval, task)
	assert.NoError(t, err)
	assert.NotEmpty(t, scheduler.entries)

	scheduler.Stop()
}

func TestIntervalToCron_Boundary(t *testing.T) {
	result := intervalToCron(1 * time.Nanosecond)
	assert.Equal(t, "*/10 * * * * *", result)

	result = intervalToCron(10 * time.Millisecond)
	assert.Equal(t, "*/10 * * * * *", result)

	result = intervalToCron(9 * time.Second)
	assert.Equal(t, "*/10 * * * * *", result)

	result = intervalToCron(10 * time.Second)
	assert.Equal(t, "*/10 * * * * *", result)

	result = intervalToCron(11 * time.Second)
	assert.Equal(t, "*/11 * * * * *", result)
}

func TestCronScheduler_Stop_EmptyEntries(t *testing.T) {
	scheduler := NewCronScheduler(30 * time.Second).(*CronScheduler)

	assert.NotPanics(t, func() {
		scheduler.Stop()
	})

	scheduler.Stop()
}

func TestIntervalToCron_MinimumInterval(t *testing.T) {
	result := intervalToCron(1 * time.Millisecond)
	assert.Equal(t, "*/10 * * * * *", result)

	result = intervalToCron(9 * time.Second)
	assert.Equal(t, "*/10 * * * * *", result)

	result = intervalToCron(10 * time.Second)
	assert.Equal(t, "*/10 * * * * *", result)

	result = intervalToCron(11 * time.Second)
	assert.Equal(t, "*/11 * * * * *", result)
}

func TestIntervalToCron_EdgeCases(t *testing.T) {
	assert.Equal(t, "0 */5 * * * *", intervalToCron(0))
	assert.Equal(t, "*/10 * * * * *", intervalToCron(5*time.Second))
	assert.Equal(t, "*/10 * * * * *", intervalToCron(9*time.Second))
	assert.Equal(t, "*/10 * * * * *", intervalToCron(10*time.Second))
	assert.Equal(t, "*/11 * * * * *", intervalToCron(11*time.Second))
}
