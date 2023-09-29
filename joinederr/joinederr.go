package joinederr

type ErrorIterator interface {
	Next() error
	HasNext() bool
}

type depthFirstUnwrapper struct {
	next       error
	nextParent []error
}

func (bfu *depthFirstUnwrapper) Next() error {
	if bfu.next == nil {
		return nil
	}

	// Split joined errors
	if x, ok := bfu.next.(interface{ Unwrap() []error }); ok {
		errs := x.Unwrap()
		if len(errs) > 0 {
			bfu.next = errs[0]
			bfu.nextParent = append(errs[1:], bfu.nextParent...)
		}
	}

	// Set return value
	r := bfu.next

	// Setup next return value
	bfu.next = nil

	if x, ok := r.(interface{ Unwrap() error }); ok {
		bfu.next = x.Unwrap()
	}

	if bfu.next == nil {
		if len(bfu.nextParent) > 0 {
			bfu.next = bfu.nextParent[0]
			bfu.nextParent = bfu.nextParent[1:]
		}
	}

	// Return
	return r
}

func (bfu *depthFirstUnwrapper) HasNext() bool {
	return bfu.next != nil
}

func NewDepthFirstIterator(err error) ErrorIterator {
	return &depthFirstUnwrapper{next: err}
}
