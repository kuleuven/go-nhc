package main

type CheckFactory func(string) (Check, error)
type Check func() (Status, string)
type Status int

const (
  OK       Status = 0
  Warning  Status = 1
  Critical Status = 2
  Unknown  Status = 3
)

func (s Status) String() string {
    if s == OK {
        return "OK"
    }
    if s == Warning {
        return "WARN"
    }
    if s == Critical {
        return "CRIT"
    }
    return "UNKNOWN"
}

func (s Status) RC() int {
    return int(s)
}
