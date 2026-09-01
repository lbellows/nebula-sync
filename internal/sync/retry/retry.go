package retry

import (
	"fmt"
	"time"

	"github.com/avast/retry-go"
	"github.com/rs/zerolog/log"

	"github.com/lovelaze/nebula-sync/internal/config"
)

const (
	AttemptsPostTeleporter = 5
	AttemptsPatchConfig    = 5
	AttemptsPostRunGravity = 5
	AttemptsPostAuth       = 3
	// FTL is often still restarting after gravity, so session teardown needs a
	// few more attempts than auth. Keep it bounded: the delay is
	// CLIENT_RETRY_DELAY_SECONDS and a failure here is only logged as a warning,
	// so a large budget just stalls the run.
	AttemptsDeleteSession = 5
)

var delay time.Duration

func Init(clientConfig *config.Client) {
	delay = time.Duration(clientConfig.RetryDelay) * time.Second
}

func Fixed(retryFunc func() error, attempts uint) error {
	return retry.Do(
		func() error {
			return retryFunc()
		},
		retry.Attempts(attempts),
		retry.Delay(delay),
		retry.LastErrorOnly(true),
		retry.DelayType(retry.FixedDelay),
		retry.OnRetry(func(n uint, err error) {
			log.Debug().Msg(fmt.Sprintf("Retrying(%d): %v", n+1, err))
		}),
	)
}
