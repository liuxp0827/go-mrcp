package mpf

import "github.com/navi-tt/go-mrcp/apr/memory"

/**
 * Create file stream.
 * @param termination the back pointer to hold
 * @param pool the pool to allocate memory from
 */
func FileStreamCreate(termination *Termination, pool *memory.AprPool) *AudioStream {
	return nil
}

/**
 * Modify file stream.
 * @param stream file stream to modify
 * @param descriptor the descriptor to modify stream according
 */
func (as *AudioStream) FileStreamModify(descriptor *AudioFileDescriptor) error {
	return nil
}