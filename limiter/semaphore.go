package limiter

type Semaphore struct {
	tokens chan struct{}
}

func NewSemaphore(limit int) *Semaphore {
	return &Semaphore{
		tokens: make(chan struct{}, limit),
	}
}

func (s *Semaphore) Acquire() bool {
	select {
	case s.tokens <- struct{}{}:
		return true
	default:
		return false
	}
}

func (s *Semaphore) Release() {
	select {
	case <-s.tokens:
	default:
	}
}
