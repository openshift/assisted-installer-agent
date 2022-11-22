package commands

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/go-openapi/swag"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/ghttp"
	"github.com/openshift/assisted-installer-agent/src/config"
	"github.com/openshift/assisted-service/models"
	"github.com/sirupsen/logrus"
)

var _ = Describe("Step processor", func() {
	var (
		ctx    context.Context
		cancel context.CancelFunc
		log    *logrus.Logger
		server *Server
		cfg    *config.AgentConfig
	)

	BeforeEach(func() {
		// Create a context with a timeout so that the test will always finish even if the
		// step processor doesn't.
		ctx, cancel = context.WithTimeout(context.Background(), 1*time.Minute)

		// Create a logger that writes to the Ginkgo output stream:
		log = logrus.New()
		log.SetOutput(GinkgoWriter)
		log.SetLevel(logrus.DebugLevel)

		// Create the mock server:
		server = NewServer()

		// Create the configuration to use the mock server:
		cfg = &config.AgentConfig{
			ConnectivityConfig: config.ConnectivityConfig{
				TargetURL: server.URL(),
			},
		}
	})

	AfterEach(func() {
		// Close the mock server:
		server.Close()
	})

	DescribeTable(
		"Retries if the service returns error response",
		func(code int) {
			// Configure the server so that the first time it responds the error code
			// and the second time it responds with the exit command.
			server.AppendHandlers(
				RespondWith(code, nil),
				RespondWithJSONEncoded(
					http.StatusOK,
					&models.Steps{
						PostStepAction: swag.String(models.StepsPostStepActionExit),
					},
				),
			)

			// Process the steps. Note that the tool runner factory is nil because we
			// will never need to execute a real command, and using nil simplifies this
			// test.
			wg := &sync.WaitGroup{}
			wg.Add(1)
			log := log.WithContext(ctx)
			ProcessSteps(ctx, cancel, cfg, nil, wg, log)
			wg.Wait()
		},
		Entry("Unauthorized (401)", http.StatusUnauthorized),
		Entry("Unavailable (503)", http.StatusServiceUnavailable),
	)

	It("Increases delay after retry", func() {
		// Configure the server so that the first three times it responds with a 503 error
		// and the fourth time it responds with the exit command, while checking that the
		// delay is increased for each attempt.
		lastTime := time.Now()
		lastDelay := time.Duration(0)
		checkHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			thisDelay := time.Since(lastTime)
			Expect(thisDelay).To(BeNumerically(">=", lastDelay))
			lastTime = time.Now()
			lastDelay = thisDelay
		})
		failHandler := RespondWith(http.StatusServiceUnavailable, nil)
		exitHandler := RespondWithJSONEncoded(
			http.StatusOK,
			&models.Steps{
				PostStepAction: swag.String(models.StepsPostStepActionExit),
			},
		)
		server.AppendHandlers(
			CombineHandlers(checkHandler, failHandler),
			CombineHandlers(checkHandler, failHandler),
			CombineHandlers(checkHandler, failHandler),
			CombineHandlers(checkHandler, exitHandler),
		)

		// Process the steps. Note that the tool runner factory is nil because we will never
		// need to execute a real command, and using nil simplifies this test.
		wg := &sync.WaitGroup{}
		wg.Add(1)
		log := log.WithContext(ctx)
		ProcessSteps(ctx, cancel, cfg, nil, wg, log)
		wg.Wait()
	})
})
