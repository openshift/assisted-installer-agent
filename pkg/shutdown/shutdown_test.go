package shutdown

import (
	"bytes"
	"context"
	"errors"
	"io"
	"syscall"
	"testing"
	"time"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
)

func TestShutdown(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Shutdown")
}

// logger is the logger used by default for the tests.
var logger *logrus.Logger

var _ = BeforeEach(func() {
	// Create a logger that writes to the Ginkgo writer, so that the log isn't mixed with the
	// regular Ginkgo output:
	logger = logrus.New()
	logger.SetLevel(logrus.DebugLevel)
	logger.SetOutput(GinkgoWriter)
})

var _ = Describe("Shutdown", func() {
	It("Can't be created without a logger", func() {
		sequence, err := NewSequence().Build()
		Expect(err).To(HaveOccurred())
		message := err.Error()
		Expect(message).To(ContainSubstring("logger"))
		Expect(message).To(ContainSubstring("mandatory"))
		Expect(sequence).To(BeNil())
	})

	It("Can't be created with a negative delay", func() {
		sequence, err := NewSequence().
			Logger(logger).
			Delay(-1 * time.Second).
			Build()
		Expect(err).To(HaveOccurred())
		message := err.Error()
		Expect(message).To(ContainSubstring("delay"))
		Expect(message).To(ContainSubstring("greater"))
		Expect(message).To(ContainSubstring("zero"))
		Expect(message).To(ContainSubstring("-1s"))
		Expect(sequence).To(BeNil())
	})

	It("Can't be created with a negative timeout", func() {
		sequence, err := NewSequence().
			Logger(logger).
			Timeout(-1 * time.Second).
			Build()
		Expect(err).To(HaveOccurred())
		message := err.Error()
		Expect(message).To(ContainSubstring("timeout"))
		Expect(message).To(ContainSubstring("greater"))
		Expect(message).To(ContainSubstring("zero"))
		Expect(message).To(ContainSubstring("-1s"))
		Expect(sequence).To(BeNil())
	})

	It("Can't be created without an exit function", func() {
		sequence, err := NewSequence().
			Logger(logger).
			Exit(nil).
			Build()
		Expect(err).To(HaveOccurred())
		message := err.Error()
		Expect(message).To(ContainSubstring("exit"))
		Expect(message).To(ContainSubstring("mandatory"))
		Expect(sequence).To(BeNil())
	})

	It("Can be created without steps", func() {
		sequence, err := NewSequence().
			Logger(logger).
			Build()
		Expect(err).ToNot(HaveOccurred())
		Expect(sequence).ToNot(BeNil())
	})

	It("Can be created with one step", func() {
		step := func(ctx context.Context) error {
			return nil
		}
		sequence, err := NewSequence().
			Logger(logger).
			Step(step).
			Build()
		Expect(err).ToNot(HaveOccurred())
		Expect(sequence).ToNot(BeNil())
	})

	It("Can be created with multiple steps", func() {
		step := func(ctx context.Context) error {
			return nil
		}
		sequence, err := NewSequence().
			Logger(logger).
			Step(step).
			Step(step).
			Step(step).
			Build()
		Expect(err).ToNot(HaveOccurred())
		Expect(sequence).ToNot(BeNil())
	})

	It("Can be created with a list of steps", func() {
		step := func(ctx context.Context) error {
			return nil
		}
		sequence, err := NewSequence().
			Logger(logger).
			Steps(step, step, step).
			Build()
		Expect(err).ToNot(HaveOccurred())
		Expect(sequence).ToNot(BeNil())
	})

	It("Executes one step", func() {
		// Create an exit function that does nothing:
		exit := func(code int) {
		}

		// Create a step function:
		called := false
		step := func(ctx context.Context) error {
			called = true
			return nil
		}

		// Create the sequence:
		sequence, err := NewSequence().
			Logger(logger).
			Step(step).
			Exit(exit).
			Build()
		Expect(err).ToNot(HaveOccurred())

		// Start the sequence:
		sequence.Start(1)

		// Verify that the step was called:
		Expect(called).To(BeTrue())
	})

	It("Executes multiple steps", func() {
		// Create an exit function that does nothing:
		exit := func(code int) {
		}

		// Create a step function:
		called1 := false
		step1 := func(ctx context.Context) error {
			called1 = true
			return nil
		}
		called2 := false
		step2 := func(ctx context.Context) error {
			called2 = true
			return nil
		}
		called3 := false
		step3 := func(ctx context.Context) error {
			called3 = true
			return nil
		}

		// Create the sequence:
		sequence, err := NewSequence().
			Logger(logger).
			Steps(step1, step2, step3).
			Exit(exit).
			Build()
		Expect(err).ToNot(HaveOccurred())

		// Start the sequence:
		sequence.Start(1)

		// Verify that the step was called:
		Expect(called1).To(BeTrue())
		Expect(called2).To(BeTrue())
		Expect(called3).To(BeTrue())
	})

	It("Executes step even if previous step failed", func() {
		// Create an exit function that does nothing:
		exit := func(code int) {
		}

		// Create a step function:
		called := false
		step1 := func(ctx context.Context) error {
			return errors.New("failed")
		}
		step2 := func(ctx context.Context) error {
			called = true
			return nil
		}

		// Create the sequence:
		sequence, err := NewSequence().
			Logger(logger).
			Steps(step1, step2).
			Exit(exit).
			Build()
		Expect(err).ToNot(HaveOccurred())

		// Start the sequence:
		sequence.Start(1)

		// Verify that the step was called:
		Expect(called).To(BeTrue())
	})

	It("Calls the exit function with the given code", func() {
		// Create an exit function that saves the exit code:
		var code int
		exit := func(c int) {
			code = c
		}

		// Create the sequence:
		sequence, err := NewSequence().
			Logger(logger).
			Exit(exit).
			Build()
		Expect(err).ToNot(HaveOccurred())

		// Start the sequence:
		sequence.Start(123)

		// Verify the code:
		Expect(code).To(Equal(123))
	})

	It("Calls the exit function with the given code even if some steps failed", func() {
		// Create an exit function that saves the exit code:
		var code int
		exit := func(c int) {
			code = c
		}

		// Create the steps:
		step1 := func(ctx context.Context) error {
			return nil
		}
		step2 := func(ctx context.Context) error {
			return errors.New("failed")
		}
		step3 := func(ctx context.Context) error {
			return nil
		}

		// Create the sequence:
		sequence, err := NewSequence().
			Logger(logger).
			Steps(step1, step2, step3).
			Exit(exit).
			Build()
		Expect(err).ToNot(HaveOccurred())

		// Start the sequence:
		sequence.Start(123)

		// Verify the code:
		Expect(code).To(Equal(123))
	})

	It("Passes a context with a deadline to the steps", func() {
		// Create an exit function that does nothing:
		exit := func(code int) {
		}

		// Create a step that verifies that the context has a deadline:
		step := func(ctx context.Context) error {
			defer GinkgoRecover()
			Expect(ctx).ToNot(BeNil())
			deadline, ok := ctx.Deadline()
			Expect(ok).To(BeTrue())
			Expect(deadline).ToNot(BeZero())
			return nil
		}

		// Create the sequence:
		sequence, err := NewSequence().
			Logger(logger).
			Step(step).
			Exit(exit).
			Build()
		Expect(err).ToNot(HaveOccurred())

		// Start the sequence:
		sequence.Start(0)
	})

	It("Cancels the context passed to steps when the timeout expires", func() {
		// Create an exit function that does nothing:
		exit := func(code int) {
		}

		// Create a step that waits longer than the timeout configured in the sequence, and
		// then verifies that the context has been cancelled:
		done := make(chan struct{})
		step := func(ctx context.Context) error {
			defer GinkgoRecover()
			defer close(done)
			time.Sleep(200 * time.Millisecond)
			err := ctx.Err()
			Expect(err).To(Equal(context.DeadlineExceeded))
			return nil
		}

		// Create the sequence, setting a timeout that will give us time to verify that the
		// context passed to the step is cancelled:
		sequence, err := NewSequence().
			Logger(logger).
			Timeout(100 * time.Millisecond).
			Step(step).
			Exit(exit).
			Build()
		Expect(err).ToNot(HaveOccurred())

		// Start the sequence:
		sequence.Start(0)

		// The step will not finish on time, so we need to explicitly wait till it finishes,
		// otherwise the expectaion failures will not be reported correctly:
		<-done
	})

	It("Waits the configured delay before running steps", func() {
		// Create an exit function that does nothing:
		exit := func(code int) {
		}

		// Create a step that does nothing:
		delay := 100 * time.Millisecond
		start := time.Now()
		step := func(ctx context.Context) error {
			defer GinkgoRecover()
			elapsed := time.Since(start)
			Expect(elapsed).To(BeNumerically(">=", delay))
			return nil
		}

		// Create the sequence, setting a timeout that will give us time to verify that the
		// context passed to the step is cancelled:
		sequence, err := NewSequence().
			Logger(logger).
			Delay(delay).
			Steps(step).
			Exit(exit).
			Build()
		Expect(err).ToNot(HaveOccurred())

		// Start the sequence:
		sequence.Start(0)
	})

	It("Aborts steps after the timeout expires", func() {
		// Create an exit function that does nothing:
		exit := func(code int) {
		}

		// Create a step that waits longer than the timeout configured in the sequence:
		step1 := func(ctx context.Context) error {
			time.Sleep(200 * time.Millisecond)
			return nil
		}

		// Create a step that should not be executed:
		called2 := false
		step2 := func(ctx context.Context) error {
			called2 = true
			return nil
		}

		// Create the sequence, setting a timeout that will give us time to verify that the
		// context passed to the step is cancelled:
		sequence, err := NewSequence().
			Logger(logger).
			Timeout(100*time.Millisecond).
			Steps(step1, step2).
			Exit(exit).
			Build()
		Expect(err).ToNot(HaveOccurred())

		// Start the sequence:
		sequence.Start(0)

		// Verify that the second step wasn't called:
		Expect(called2).To(BeFalse())
	})

	It("Writes step failure to the log", func() {
		// Create an exit function that does nothing:
		exit := func(code int) {
		}

		// Create a step that fails with an error easy to locate in the log:
		key := uuid.New().String()
		step := func(ctx context.Context) error {
			return errors.New(key)
		}

		// Create a logger that writes to the Ginkgo writer and also to a memory buffer, so
		// that we can inspect it:
		buffer := &bytes.Buffer{}
		writer := io.MultiWriter(GinkgoWriter, buffer)
		multi := logrus.New()
		multi.SetLevel(logrus.DebugLevel)
		multi.SetOutput(writer)

		// Create the sequence:
		sequence, err := NewSequence().
			Logger(multi).
			Step(step).
			Exit(exit).
			Build()
		Expect(err).ToNot(HaveOccurred())

		// Start the sequence:
		sequence.Start(0)

		// Verify that the log contains the step error:
		Expect(buffer.String()).To(ContainSubstring(key))
	})

	It("Can have one step added after the initial creation", func() {
		// Create an exit function that does nothing:
		exit := func(code int) {
		}

		// Create a step that fails with an error easy to locate in the log:
		called1 := false
		step1 := func(ctx context.Context) error {
			called1 = true
			return nil
		}
		called2 := false
		step2 := func(ctx context.Context) error {
			called2 = true
			return nil
		}

		// Create the sequence with the first step:
		sequence, err := NewSequence().
			Logger(logger).
			Steps(step1).
			Exit(exit).
			Build()
		Expect(err).ToNot(HaveOccurred())

		// Add the second step:
		sequence.AddStep(step2)

		// Start the sequence:
		sequence.Start(0)

		// Verify that both steps have been called:
		Expect(called1).To(BeTrue())
		Expect(called2).To(BeTrue())
	})

	It("Can have multiple steps added after the initial creation", func() {
		// Create an exit function that does nothing:
		exit := func(code int) {
		}

		// Create a step that fails with an error easy to locate in the log:
		called1 := false
		step1 := func(ctx context.Context) error {
			called1 = true
			return nil
		}
		called2 := false
		step2 := func(ctx context.Context) error {
			called2 = true
			return nil
		}
		called3 := false
		step3 := func(ctx context.Context) error {
			called3 = true
			return nil
		}

		// Create the sequence with the first step:
		sequence, err := NewSequence().
			Logger(logger).
			Steps(step1).
			Exit(exit).
			Build()
		Expect(err).ToNot(HaveOccurred())

		// Add the additional steps:
		sequence.AddSteps(step2, step3)

		// Start the sequence:
		sequence.Start(0)

		// Verify that both steps have been called:
		Expect(called1).To(BeTrue())
		Expect(called2).To(BeTrue())
		Expect(called3).To(BeTrue())
	})

	It("Calls the exit function even if steps take longer than the timeout", func() {
		// Create an exit function that does nothing:
		exit := func(code int) {
		}

		// Create a step that takes longer than the timeout:
		timeout := 100 * time.Millisecond
		step := func(ctx context.Context) error {
			time.Sleep(2 * timeout)
			return nil
		}

		// Create the sequence:
		sequence, err := NewSequence().
			Logger(logger).
			Timeout(timeout).
			Step(step).
			Exit(exit).
			Build()
		Expect(err).ToNot(HaveOccurred())

		// Start the sequence:
		start := time.Now()
		sequence.Start(0)

		// Verify that this took approximately the specifed timeout.Note that this will
		// never be exact, because the sequence takes additional time to do its work, that
		// is the reason for the 20 milliseconds of margin.
		elapsed := time.Since(start)
		Expect(elapsed).To(BeNumerically("==", timeout, 20*time.Millisecond))
	})

	It("Does nothing the second time it is started", func() {
		// Create an exit function that saves the code:
		var code int
		exit := func(c int) {
			code = c
		}

		// Create a step that increases a counter each time it is called:
		count := 0
		step := func(ctx context.Context) error {
			count++
			return nil
		}

		// Create the sequence:
		sequence, err := NewSequence().
			Logger(logger).
			Step(step).
			Exit(exit).
			Build()
		Expect(err).ToNot(HaveOccurred())

		// Start the sequence twice, each time with a different code:
		sequence.Start(123)
		sequence.Start(234)

		// Verify that the exit function was called only with the first code, and that the
		// step was called only once:
		Expect(code).To(Equal(123))
		Expect(count).To(Equal(1))
	})

	It("Starts automatically when it receives a signal", func() {
		// Create an exit function that does nothing:
		exit := func(c int) {
		}

		// Create a step:
		done := make(chan struct{})
		step := func(ctx context.Context) error {
			close(done)
			return nil
		}

		// Create the sequence:
		_, err := NewSequence().
			Logger(logger).
			Signal(syscall.SIGUSR1).
			Step(step).
			Exit(exit).
			Build()
		Expect(err).ToNot(HaveOccurred())

		// Send the signal:
		err = syscall.Kill(syscall.Getpid(), syscall.SIGUSR1)
		Expect(err).ToNot(HaveOccurred())

		// Verify that the step was executed:
		Eventually(done).Should(BeClosed())
	})
})
