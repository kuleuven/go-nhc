package main

type CheckFactory func(string) (Check, error)
type Check func() (Status, string)
type Status int

const (
	OK       Status = 0
	Warning  Status = 1
	Critical Status = 2
	Unknown  Status = 3
	Ignore   Status = -1
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
	if s == Ignore {
		return "IGNORE"
	}
	return "UNKNOWN"
}

func (s Status) RC() int {
	return int(s)
}

func (s Status) IsFatal() bool {
	return s != OK && s != Ignore && s != Warning
}

// Translate a Status in a non-fatal one
func (s Status) NonFatal() Status {
	if !s.IsFatal() {
		return s
	} else if s == Critical {
		return Warning
	} else {
		return Ignore
	}
}

// Translate a Status in a non-fatal one if mayBeFatal is false
func (s Status) NonFatalUnless(mayBeFatal bool) Status {
	if mayBeFatal {
		return s
	}
	return s.NonFatal()
}

// Compare two - order Critical > Warning > Unknown > OK > Ignore
func (s Status) Compare(t Status) int {
	if s == t {
		return 0
	}
	if t == Critical {
		return -1
	}
	if s == Critical {
		return 1
	}
	if t == Warning {
		return -1
	}
	if s == Warning {
		return 1
	}
	if s < t {
		return -1
	} else {
		return 1
	}
}
