## Timers

Timers are represented by the time that the timer has started.

Timers have three states:

- Running:

      time.Since(started) < duration

  Running timers, upon expiry, typically move into `inactive` state.

- Inactive/Disabled:

      time.IsZero(started)

  Inactive timers are started by setting `started` to `time.Now()`.

- Elapsed:

      started.Equal(elapsed)

  where `elapsed` is a sentinel value of `Unix(0,1)`.
  Elapsed timers are considered to have expired.
