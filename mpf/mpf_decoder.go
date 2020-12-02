package mpf

import "bytes"

type Decoder struct {
	Base    *AudioStream
	Source  *AudioStream
	Codec   *Codec
	FrameIn Frame
}

func DecoderDestroy(stream *AudioStream) error {
	decoder := stream.Obj.(*Decoder)
	return AudioStreamDestroy(decoder.Source)
}

func DecoderOpen(stream *AudioStream, codec *Codec) error {
	decoder := stream.Obj.(*Decoder)
	err := DecoderOpen(stream, decoder.Codec)
	if err != nil {
		return err
	}
	return decoder.Source.AudioStreamRXOpen(decoder.Codec)
}

func DecoderClose(stream *AudioStream) error {
	decoder := stream.Obj.(*Decoder)
	err := decoder.Codec.CodecClose()
	if err != nil {
		return err
	}
	return decoder.Source.AudioStreamRXClose()
}

func DecoderProcess(stream *AudioStream, frame *Frame) error {
	decoder := stream.Obj.(*Decoder)
	decoder.FrameIn.Type = MEDIA_FRAME_TYPE_NONE
	decoder.FrameIn.Marker = MPF_MARKER_NONE

	if err := decoder.Source.AudioStreamFrameRead(&decoder.FrameIn); err != nil {
		return err
	}

	frame.Type = decoder.FrameIn.Type
	frame.Marker = decoder.FrameIn.Marker
	if (frame.Type & MEDIA_FRAME_TYPE_EVENT) == MEDIA_FRAME_TYPE_EVENT {
		frame.EventFrame = decoder.FrameIn.EventFrame
	}
	if (frame.Type & MEDIA_FRAME_TYPE_AUDIO) == MEDIA_FRAME_TYPE_AUDIO {
		return decoder.Codec.CodecDecode(&decoder.FrameIn.CodecFrame, &frame.CodecFrame)
	}
	return nil
}

/**
 * Create audio stream decoder.
 * @param source the source to get encoded stream from
 * @param codec the codec to use for decode
 * @param pool the pool to allocate memory from
 */
func DecoderCreate(source *AudioStream, codec *Codec) *AudioStream {
	if source == nil || codec == nil {
		return nil
	}

	var vtable = AudioStreamVTable{
		Destroy:    DecoderDestroy,
		OpenRX:     DecoderOpen,
		CloseRX:    DecoderClose,
		ReadFrame:  DecoderProcess,
		OpenTX:     nil,
		CloseTX:    nil,
		WriteFrame: nil,
		Trace:      nil,
	}

	decoder := new(Decoder)
	capabilities := StreamCapabilitiesCreate(STREAM_DIRECTION_RECEIVE)
	decoder.Base = AudioStreamCreate(decoder, &vtable, capabilities)
	if decoder.Base == nil {
		return nil
	}
	decoder.Base.RXDescriptor = CodecLPcmDescriptorCreate(source.RXDescriptor.SamplingRate, source.RXDescriptor.ChannelCount)
	decoder.Base.RXEventDescriptor = source.RXEventDescriptor

	decoder.Source = source
	decoder.Codec = codec

	frameSize := source.RXDescriptor.CodecFrameSizeCalculate(codec.Attribs)
	decoder.FrameIn.CodecFrame.Size = frameSize
	decoder.FrameIn.CodecFrame.Buffer = bytes.NewBuffer(make([]byte, 0))

	return decoder.Base
}
