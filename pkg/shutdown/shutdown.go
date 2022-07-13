package shutdown

import (
	"context"
	"errors"
	"fmt"
	"os"
	sgn "os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"
)

// SequenceBuilder contains the data and logic needed to create a shutdown sequence. Don't create
// instances of this type directly, use the NewSequence function instead.
type SequenceBuilder struct {
	logger  *logrus.Logger
	delay   time.Duration
	timeout time.Duration
	signals []os.Signal
	steps   []func(context.Context) error
	exit    func(int)
}

// Sequence is a shutdown sequence. Don't create instances of this type directly, use the
// NewSequence function instead.
type Sequence struct {
	logger  *logrus.Logger
	delay   time.Duration
	timeout time.Duration
	steps   []func(context.Context) error
	exit    func(int)
	lock    *sync.Mutex
	done    bool
}

// NewSequence creates a builder that can then be used to configure create a sequence builder.
//
// A sequence contains a list of steps, a timeout and an exit function. The list of steps are
// functions that will be called in order when the sequence is started (when the Start method is
// called). The exit function will be called after all the steps have finished, or when the
// configured timeout expires, whatever happens first.
//
// The step functions receive a context as parameter, and return an error. For example, a step that
// removes temporary directory could look be added line this:
//
//	// Perform some initialization task that requires a temporary directory:
//	tmp, err := uiutil.Tempdir("", "*.tmp")
//	if err != nil {
//		...
//	}
//
//	// Create the shutdown sequence, and remember to remove the temporary directory:
//	sequence, err := shutdown.NewSequence().
//		Logger(logger).
//		Step(func (ctx context.Context) error {
//			return os.RemoveAll(tmp)
//		}).
//		Build()
//	if err != nil {
//		...
//	}
//
//	// Eventually start the shutdown sequence:
//	squence.Start(0)
//
// Steps can also be added after the initial creation of the sequence:
//
//	// someFunc receives the shutdown sequence that was configured and created somewhere
//	// else.
//	func someFunc(shutdown *shutdown.Sequence) error {
//		// Create a temporary directory:
//		tmp, err := uiutil.Tempdir("", "*.tmp")
//		if err != nil {
//			return error
//		}
//
//		// Remember to delete the directory during shutdown:
//		squence.AddStep(func (ctx context.Context) error {
//			return os.RemoveAll(tmp)
//		}
//		...
//	}
//
// The default exit function is os.Exit, and there is usually no need to change it. If needed it can
// be changed using the Exit method of the builder. For example, in unit tests it is convenient to
// avoid exiting the process:
//
//	sequence, err := shutdown.NewSequence().
//		Logger(logger).
//		Exit(func (code int) {)).
//		Build()
//	if err != nil {
//		...
//	}
//
// The steps run in a separate goroutine and with a context that has the timeout set with the
// Timeout method of the builder (1 minute by default). If a step takes longer than that the rest of
// the steps will be skipped, but the exit function will be called anyhow, which will usually means
// that the process will exit anyhow.
//
// Note that steps are responsible for honouring the timeout set in the context, the sequence can't
// and will not stop such goroutine, because there is no way to do that in Go. It will however call
// the exit function, and if it is the default os.Exit it will kill the process and therefore all
// goroutines.
//
// When the Start method is called the shutdown sequence will start inmediately unless a delay has
// been specified in the configuration (using the Delay method of the builder). This is intended for
// situations where some shutdown step is started outside of the control of the sequence.
//
// The sequence can also be configured to automatically start when certain signals are received.
// For example in Unix systems you will probably want to start the sequence when the SIGKILL or
// SIGTERM signals are received:
//
//	sequence, err := shutdown.NewSequence().
//		Logger(logger).
//		Signals(syscall.SIGKILL, syscall.SIGTERM).
//		Steps(...).
//		Build()
//	if err != nil {
//		...
//	}
func NewSequence() *SequenceBuilder {
	return &SequenceBuilder{
		delay:   0,
		timeout: 1 * time.Minute,
		exit:    os.Exit,
	}
}

// Logger sets the logger that the shutdown sequence will use to write messagse to the log. This is
// mandatory.
func (b *SequenceBuilder) Logger(value *logrus.Logger) *SequenceBuilder {
	b.logger = value
	return b
}

// Delay sets the time that the shutdown sequence will be delayed after the Start method is called.
// It is intended for situations where some shutdown steps can't be added to the sequence properly.
// This is optional and the default value is zero.
func (b *SequenceBuilder) Delay(value time.Duration) *SequenceBuilder {
	b.delay = value
	return b
}

// Timeout sets the maximum time that will pass between the call to the Start method and the call to
// the exit function. Steps that don't complete in that time will simple be ignored. This optional
// and the default value is one minute.
func (b *SequenceBuilder) Timeout(value time.Duration) *SequenceBuilder {
	b.timeout = value
	return b
}

// Signal adds a signal that will start the sequence. This is optional, and by default no signal is
// used.
func (b *SequenceBuilder) Signal(value os.Signal) *SequenceBuilder {
	b.signals = append(b.signals, value)
	return b
}

// Signals adds a list of signals that will start the sequence. This is optional and by default no
// signal is used.
func (b *SequenceBuilder) Signals(values ...os.Signal) *SequenceBuilder {
	b.signals = append(b.signals, values...)
	return b
}

// Step adds an step to the sequence.
func (b *SequenceBuilder) Step(value func(context.Context) error) *SequenceBuilder {
	b.steps = append(b.steps, value)
	return b
}

// Steps adds a list of steps to the sequence.
func (b *SequenceBuilder) Steps(values ...func(context.Context) error) *SequenceBuilder {
	b.steps = append(b.steps, values...)
	return b
}

// Exit sets the function that will be called when the sequence has been completed. The default is
// to call os.Exit. There is usually no need to set this explicitly, it is intended for unit tests
// where exiting the process isn't convenient.
func (b *SequenceBuilder) Exit(value func(int)) *SequenceBuilder {
	b.exit = value
	return b
}

// Build uses the data stored in the builder to create a new shutdown sequence. Note that this
// doesn't start the shutdown sequence, to do that use the Start method.
func (b *SequenceBuilder) Build() (result *Sequence, err error) {
	// Check parameters:
	if b.logger == nil {
		err = errors.New("logger is mandatory")
		return
	}
	if b.delay < 0 {
		err = fmt.Errorf(
			"delay must be greater or equal than zero, but it is %s",
			b.delay,
		)
		return
	}
	if b.timeout < 0 {
		err = fmt.Errorf(
			"timeout must be greater or equal than zero, but it is %s",
			b.timeout,
		)
		return
	}
	if b.exit == nil {
		err = errors.New("exit function is mandatory")
		return
	}

	// Create a logger for the sequence with some additional information:
	logger := b.logger.WithFields(logrus.Fields{
		"name":    "shutdown",
		"delay":   b.delay,
		"timeout": b.timeout,
	})

	// Copy the steps to avoid potential side effects if the builder is changed after creating
	// the object:
	steps := make([]func(context.Context) error, len(b.steps))
	copy(steps, b.steps)

	// Create and populate the object:
	result = &Sequence{
		logger:  b.logger,
		delay:   b.delay,
		timeout: b.timeout,
		steps:   b.steps,
		exit:    b.exit,
		lock:    &sync.Mutex{},
		done:    false,
	}

	// Start the sequence automatically when one of the configured signals is received:
	signals := make(chan os.Signal, 1)
	for _, signal := range b.signals {
		sgn.Notify(signals, signal)
	}
	go func() {
		signal := <-signals
		var name string
		switch typed := signal.(type) {
		case syscall.Signal:
			name = unix.SignalName(typed)
		default:
			name = signal.String()
		}
		logger.WithFields(logrus.Fields{
			"signal": name,
		}).Info("Shutdown sequence started by signal")
		result.Start(0)
	}()

	return
}

// AddStep add one step to the sequence.
func (s *Sequence) AddStep(step func(context.Context) error) {
	s.steps = append(s.steps, step)
}

// AddSteps adds a list of steps to the sequence.
func (s *Sequence) AddSteps(steps ...func(context.Context) error) {
	s.steps = append(s.steps, steps...)
}

// Star starts the shutdown sequence. It will run all the steps in the same order that they were
// added, and then call the exit function (by default os.Exit) with the given code.
func (s *Sequence) Start(code int) {
	// Make sure that we never run the sequence multiple times simultaneously:
	s.lock.Lock()
	defer s.lock.Unlock()

	// Create a logger with some additional information about the sequence:
	logger := s.logger.WithFields(logrus.Fields{
		"delay":   s.delay,
		"timeout": s.timeout,
		"code":    code,
	})

	// Check if the sequence has already been completed:
	if s.done {
		logger.Error(
			"Shutdown has been requested again after it was already completed, this " +
				"is most likely a bug in the code that uses it, will do nothing",
		)
		return
	}

	// Wait a bit before starting the sequence:
	if s.delay > 0 {
		logger.Info("Shutdown has been requested and will start after the delay")
		time.Sleep(s.delay)
	}

	// Create a context with the configured timeout:
	deadline := time.Now().Add(s.timeout)
	ctx, cancel := context.WithDeadline(context.Background(), deadline)
	defer cancel()

	// Add the deadline to the logger:
	logger = logger.WithField("deadline", deadline.UTC().Format(time.RFC3339))

	// Run the steps in order in a separate goroutine:
	logger.Info("Shutdown sequence started")
	go func() {
	loop:
		for i, step := range s.steps {
			stepLogger := logger.WithFields(logrus.Fields{
				"step": i,
			})
			stepLogger.Info("Starting shutdown step")
			err := step(ctx)
			if err != nil {
				stepLogger.WithError(err).Error("Shutdown step failed")
			} else {
				stepLogger.Info("Shutdown step succeeded")
			}
			select {
			case <-ctx.Done():
				remaining := len(s.steps) - i - 1
				if remaining > 0 {
					stepLogger.Info(
						"Remaining shutdown steps aborted due to timeout",
					)
					break loop
				}
			default:
				continue loop
			}
		}
		cancel()
	}()

	// Wait for the steps to finish or for the timeout to expire, whatever happens first:
	logger.Info("Shutdown sequence waiting for steps to finish")
	<-ctx.Done()

	// Mark the sequence as done and exit:
	logger.Info("Shutdown sequence finished, exiting")
	s.done = true
	s.exit(code)
}
