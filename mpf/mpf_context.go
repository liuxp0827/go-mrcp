package mpf

import (
	"container/list"
	"fmt"
	"github.com/navi-tt/go-mrcp/apr"
)

/** Factory of media contexts */
type ContextFactory struct {
	Link *list.List
	/** Ring head */
	Head *list.Element // List of header fields (name-value pairs), Ring 的 Value 就是 *AptHeaderField head;
}

/** Item of the association matrix */
type MatrixItem struct {
	On uint8
}

/** Item of the association matrix header */
type HeaderItem struct {
	termination *Termination
	TXCount     uint8
	RXCount     uint8
}

/** Media processing context */
type Context struct {

	/** Ring entry */
	Element *list.Element
	/** Back pointer to the context factory */
	Factory *ContextFactory

	/** Informative name of the context used for debugging */
	Name string
	/** External object */
	Obj interface{}

	/** Max number of terminations in the context */
	Capacity int64
	/** Current number of terminations in the context */
	Count int64
	/** Header of the association matrix */
	header []HeaderItem
	/** Association matrix, which represents the topology */
	matrix [][]MatrixItem

	/** Array of media processing objects constructed while
	  applying topology based on association matrix */
	mpfObjects *apr.ArrayHeader
}

/**
 * Create factory of media contexts.
 */
func ContextFactoryCreate() *ContextFactory {
	return &ContextFactory{
		Link: list.New(),
		Head: nil,
	}
}

/**
 * Destroy factory of media contexts.
 */
func ContextFactoryDestroy(factory *ContextFactory) error {
	for factory.Link.Len() > 0 {
		head := factory.Link.Front()
		ctx := head.Value.(*Context)
		_ = ContextDestroy(ctx)
		factory.Link.Remove(head)
	}
	return nil
}

/**
 * Process factory of media contexts.
 */
func ContextFactoryProcess(factory *ContextFactory) error {
	head := factory.Link.Front()
	if head == nil {
		return nil
	}
	ctx := head.Value.(*Context)
	for ctx != nil {
		_ = ctx.ContextProcess()
		head = head.Next()
		if head != nil {
			ctx = head.Value.(*Context)
		} else {
			ctx = nil
		}
	}
	return nil
}

/**
 * Create MPF context.
 * @param factory the factory context belongs to
 * @param name the informative name of the context
 * @param obj the external object associated with context
 * @param max_termination_count the max number of terminations in context
 * @param pool the pool to allocate memory from
 */
func (f *ContextFactory) ContextCreate(name string, obj interface{}, maxTerminationCount int64) *Context {
	var (
		matrixItem *MatrixItem
		headerItem *HeaderItem
		context    = &Context{
			Factory:    f,
			Name:       name,
			Obj:        obj,
			Capacity:   maxTerminationCount,
			Count:      0,
			header:     make([]HeaderItem, maxTerminationCount),
			matrix:     make([][]MatrixItem, maxTerminationCount),
			mpfObjects: apr.NewArrayHeader(1),
		}
	)
	for i := int64(0); i < context.Capacity; i++ {
		headerItem = &context.header[i]
		headerItem.termination = nil
		headerItem.TXCount = 0
		headerItem.RXCount = 0
		context.matrix[i] = make([]MatrixItem, context.Capacity)
		for j := int64(0); j < context.Capacity; j++ {
			matrixItem = &context.matrix[i][j]
			matrixItem.On = 0
		}
	}
	ele := f.Link.PushBack(context)
	context.Element = ele
	return context
}

/**
 * Destroy MPF context.
 * @param context the context to destroy
 */
func ContextDestroy(context *Context) error {
	for i := int64(0); i < context.Capacity; i++ {
		termination := context.header[i].termination
		if termination != nil {
			_ = context.ContextTerminationSubtract(termination)

			err := termination.TerminationSubtract()
			if err != nil {
				return err
			}
		}
	}
	return nil
}

/**
 * Get external object associated with MPF context.
 * @param context the context to get object from
 */
func (context *Context) ContextObjectGet() interface{} {
	return context.Obj
}

/**
 * Add termination to context.
 * @param context the context to add termination to
 * @param termination the termination to add
 */
func (context *Context) ContextTerminationAdd(termination *Termination) bool {
	for i := int64(0); i < context.Capacity; i++ {
		headerItem := &context.header[i]
		if headerItem.termination != nil {
			continue
		}
		if context.Count == 0 {
			context.Factory.Link.PushBack(context)
		}

		headerItem.termination = termination
		headerItem.TXCount = 0
		headerItem.RXCount = 0

		termination.slot = i
		context.Count++
		return true
	}
	return false
}

/**
 * Subtract termination from context.
 * @param context the context to subtract termination from
 * @param termination the termination to subtract
 */
func (context *Context) ContextTerminationSubtract(termination *Termination) bool {
	var (
		i    = termination.slot
		j, k int64
	)
	if i >= context.Capacity {
		return false
	}
	headerItem1 := &context.header[i]
	if headerItem1.termination != termination {
		return false
	}

	for j, k = 0, 0; j < context.Capacity && k < context.Count; j++ {
		headerItem2 := &context.header[j]
		if headerItem2.termination != nil {
			continue
		}
		k++

		item := &context.matrix[i][j]
		if item.On > 0 {
			item.On = 0
			headerItem1.TXCount--
			headerItem2.RXCount--
		}

		item = &context.matrix[j][i]
		if item.On > 0 {
			item.On = 0
			headerItem2.TXCount--
			headerItem1.RXCount--
		}
	}
	headerItem1.termination = nil
	termination.slot = -1
	context.Count--

	if context.Count <= 0 {
		context.Factory.Link.Remove(context.Element)
	}

	return true
}

/**
 * Add association between specified terminations.
 * @param context the context to add association in the scope of
 * @param termination1 the first termination to associate
 * @param termination2 the second termination to associate
 */
func (context *Context) ContextAssociationAdd(termination1, termination2 *Termination) error {
	var (
		i, j = termination1.slot, termination2.slot
	)

	if i >= context.Capacity || j >= context.Capacity {
		return fmt.Errorf("slot >= context.Capacity")
	}

	headerItem1 := &context.header[i]
	headerItem2 := &context.header[j]

	if headerItem1.termination != termination1 || headerItem2.termination != termination2 {
		return fmt.Errorf("no match Termination")
	}

	matrixItem1 := &context.matrix[i][j]
	matrixItem2 := &context.matrix[j][i]

	/* 1 . 2 */
	if matrixItem1.On <= 0 {
		if StreamDirectionCompatibilityCheck(headerItem1.termination, headerItem2.termination) {
			matrixItem1.On = 1
			headerItem1.TXCount++
			headerItem2.RXCount++
		}
	}

	/* 2 . 1 */
	if matrixItem2.On <= 0 {
		if StreamDirectionCompatibilityCheck(headerItem2.termination, headerItem1.termination) {
			matrixItem2.On = 1
			headerItem2.TXCount++
			headerItem1.RXCount++
		}
	}
	return nil
}

/**
 * Remove association between specified terminations.
 * @param context the context to remove association in the scope of
 * @param termination1 the first termination
 * @param termination2 the second termination
 */
func (context *Context) ContextAssociationRemove(termination1, termination2 *Termination) error {
	var (
		i, j = termination1.slot, termination2.slot
	)

	if i >= context.Capacity || j >= context.Capacity {
		return fmt.Errorf("slot >= context.Capacity")
	}

	headerItem1 := &context.header[i]
	headerItem2 := &context.header[j]

	if headerItem1.termination != termination1 || headerItem2.termination != termination2 {
		return fmt.Errorf("no match Termination")
	}

	matrixItem1 := &context.matrix[i][j]
	matrixItem2 := &context.matrix[j][i]

	/* 1 . 2 */
	if matrixItem1.On > 0 {
		matrixItem1.On = 0
		headerItem1.TXCount--
		headerItem2.RXCount--
	}

	/* 2 . 1 */
	if matrixItem2.On > 0 {
		matrixItem2.On = 0
		headerItem2.TXCount--
		headerItem1.RXCount--
	}
	return nil
}

/**
 * Reset assigned associations and destroy applied topology.
 * @param context the context to reset associations for
 */
func (context *Context) ContextAssociationsReset() error {
	var (
		i, j, k int64
	)
	/* destroy existing topology / if any */
	_ = ContextDestroy(context)

	/* reset assigned associations */
	for ; i < context.Capacity && k < context.Count; i++ {
		headerItem1 := &context.header[i]
		if headerItem1.termination == nil {
			continue
		}
		k++

		if headerItem1.TXCount <= 0 && headerItem1.RXCount <= 0 {
			continue
		}

		for j = i; j < context.Capacity; j++ {
			headerItem2 := &context.header[j]
			if headerItem2.termination == nil {
				continue
			}
			item := &context.matrix[i][j]
			if item.On > 0 {
				item.On = 0
				headerItem1.TXCount--
				headerItem2.RXCount--
			}
			item = &context.matrix[j][i]
			if item.On > 0 {
				item.On = 0
				headerItem2.TXCount--
				headerItem1.RXCount--
			}
		}
	}

	return nil
}

func (context *Context) ContextObjectAdd(object *Object) error {
	if object == nil {
		return fmt.Errorf("object is nil")
	}
	context.mpfObjects.Stack.Push(object)
	return nil
}

/**
 * Apply topology.
 * @param context the context to apply topology for
 */
func (context *Context) ContextTopologyApply() error {
	/* first destroy existing topology / if any */
	_ = context.ContextTopologyDestroy()

	var (
		i, k   int64
		object *Object
		err    error
	)
	for ; i < context.Capacity && k < context.Count; i++ {
		headerItem := &context.header[i]
		if headerItem.termination == nil {
			continue
		}
		k++

		if headerItem.TXCount > 0 {
			if headerItem.TXCount == 1 {
				object, err = context.ContextBridgeCreate(i)
				if err != nil {
					return err
				}
			} else {
				object, err = context.ContextMultiplierCreate(i)
				if err != nil {
					return err
				}
			}

			err = context.ContextObjectAdd(object)
			if err != nil {
				return err
			}
		}

		if headerItem.RXCount > 1 {
			object, err = context.ContextMixerCreate(i)
			if err != nil {
				return err
			}
			err = context.ContextObjectAdd(object)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

/**
 * Destroy topology.
 * @param context the context to destroy topology for
 */
func (context *Context) ContextTopologyDestroy() error {

	if context.mpfObjects != nil && !context.mpfObjects.Stack.IsEmpty() {
		for i := 0; i < context.mpfObjects.Stack.Size(); i++ {
			object := context.mpfObjects.ArrayHeaderIndex(i).(*Object)
			_ = ObjectDestroy(object)
		}
		context.mpfObjects.Stack.Clear()
	}
	return nil
}

/**
 * Process context.
 * @param context the context to process
 */
func (context *Context) ContextProcess() error {
	if context.mpfObjects != nil && !context.mpfObjects.Stack.IsEmpty() {
		for i := 0; i < context.mpfObjects.Stack.Size(); i++ {
			object := context.mpfObjects.ArrayHeaderIndex(i).(*Object)
			if object != nil && object.Process != nil {
				err := object.Process(object)
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (context *Context) ContextBridgeCreate(i int64) (*Object, error) {
	var (
		headerItem1 = &context.header[i]
		headerItem2 *HeaderItem
		item        *MatrixItem
		j           int64
	)

	for ; j < context.Capacity; j++ {
		headerItem2 = &context.header[j]
		if headerItem2.termination == nil {
			continue
		}
		item = &context.matrix[i][j]
		if item.On <= 0 {
			continue
		}

		if headerItem2.RXCount > 1 {
			/* mixer will be created instead */
			return nil, nil
		}

		/* create bridge i -> j */

		if headerItem1.termination != nil && headerItem2.termination != nil {
			return BridgeCreate(headerItem1.termination.audioStream,
				headerItem2.termination.audioStream,
				headerItem1.termination.codecManager,
				context.Name)
		}
	}
	return nil, nil
}

func (context *Context) ContextMultiplierCreate(i int64) (*Object, error) {
	var (
		headerItem1 = &context.header[i]
		headerItem2 *HeaderItem
		item        *MatrixItem
		j, k        int64
		sinkArr     = make([]*AudioStream, headerItem1.TXCount)
	)

	for ; j < context.Capacity && k < int64(headerItem1.TXCount); j++ {
		headerItem2 = &context.header[j]
		if headerItem2.termination == nil {
			continue
		}
		item = &context.matrix[i][j]
		if item.On <= 0 {
			continue
		}
		sinkArr[k] = headerItem2.termination.audioStream
		k++
	}
	return MultiplierCreate(headerItem1.termination.audioStream,
		sinkArr, int64(len(sinkArr)), headerItem1.termination.codecManager, context.Name), nil
}

func (context *Context) ContextMixerCreate(j int64) (*Object, error) {
	var (
		headerItem1 = &context.header[j]
		headerItem2 *HeaderItem
		item        *MatrixItem
		i, k        int64
		sourceArr   = make([]*AudioStream, headerItem1.RXCount)
	)

	for ; i < context.Capacity && k < int64(headerItem1.RXCount); i++ {
		headerItem2 = &context.header[i]
		if headerItem2.termination == nil {
			continue
		}
		item = &context.matrix[i][j]
		if item.On <= 0 {
			continue
		}
		sourceArr[k] = headerItem2.termination.audioStream
		k++
	}
	return MixerCreate(sourceArr, int64(len(sourceArr)), headerItem1.termination.audioStream, headerItem1.termination.codecManager, context.Name), nil
}

func StreamDirectionCompatibilityCheck(termination1, termination2 *Termination) bool {
	var (
		source = termination1.audioStream
		sink   = termination2.audioStream
	)
	if source != nil && (source.direction&STREAM_DIRECTION_RECEIVE) == STREAM_DIRECTION_RECEIVE &&
		sink != nil && (sink.direction&STREAM_DIRECTION_SEND) == STREAM_DIRECTION_SEND {
		return true
	}
	return false
}
