package mpf

import (
	"math"
	"sync"
)

/** DTMF detector band */
type DtmfDetectorBand = int

const (
	/** Detect tones in-band */
	MPF_DTMF_DETECTOR_INBAND DtmfDetectorBand = 0x1
	/** Detect named events out-of-band */
	MPF_DTMF_DETECTOR_OUTBAND DtmfDetectorBand = 0x2
	/** Detect both in-band and out-of-band digits */
	MPF_DTMF_DETECTOR_BOTH = MPF_DTMF_DETECTOR_INBAND | MPF_DTMF_DETECTOR_OUTBAND
)

/** Max detected DTMF digits buffer length */
const MPF_DTMFDET_BUFFER_LEN = 32

/** Number of DTMF frequencies */
const DTMF_FREQUENCIES = 8

/** Window length in samples (at 8kHz) for Goertzel's frequency analysis */
const GOERTZEL_SAMPLES_8K = 102

/** See RFC4733 */
const DTMF_EVENT_ID_MAX = 15 /* 0123456789*#ABCD */

/** Media Processing Framework's Dual Tone Multiple Frequency detector */
type DtmfDetector struct {

	/** Mutex to guard the buffer */
	mutex sync.Mutex
	/** Recognizer band */
	Band DtmfDetectorBand
	/** Detected digits buffer */
	buf [MPF_DTMFDET_BUFFER_LEN + 1]byte
	/** Number of digits in the buffer */
	Digits int64
	/** Number of lost digits due to full buffer */
	LostDigits int64
	/** Frequency analyzators */
	energies [DTMF_FREQUENCIES]GoertzelState
	/** Total energy of signal */
	TotalEnergy float64
	/** Number of samples in a window */
	WSamples int64
	/** Number of samples processed */
	NSamples int64
	/** Previously detected and last reported digits */
	last1, last2, curr byte
}

/**
 * Goertzel frequency detector (second-order IIR filter) state:
 *
 * s(t) = x(t) + coef * s(t-1) - s(t-2), where s(0)=0; s(1) = 0;
 * x(t) is the input signal
 *
 * Then energy of frequency f in the signal is:
 * X(f)X'(f) = s(t-2)^2 + s(t-1)^2 - coef*s(t-2)*s(t-1)
 */
type GoertzelState struct {
	/** coef = cos(2*pi*f_tone/f_sampling) */
	Coef float64
	/** s(t-2) or resulting energy @see goertzel_state_t */
	S1 float64
	/** s(t-1) @see goertzel_state_t */
	S2 float64
}

/** DTMF frequencies */
var DtmfFreqs = [DTMF_FREQUENCIES]float64{
	697, 770, 852, 941, /* Row frequencies */
	1209, 1336, 1477, 1633} /* Col frequencies */

/** [row, col] major frequency to digit mapping */
var freq2Digits = [DTMF_FREQUENCIES / 2][DTMF_FREQUENCIES / 2]byte{
	{'1', '2', '3', 'A'},
	{'4', '5', '6', 'B'},
	{'7', '8', '9', 'C'},
	{'*', '0', '#', 'D'},
}

/**
 * Create MPF DTMF detector (advanced).
 * @param stream      A stream to get digits from.
 * @param band        One of:
 *   - MPF_DTMF_DETECTOR_INBAND: detect audible tones only
 *   - MPF_DTMF_DETECTOR_OUTBAND: detect out-of-band named-events only
 *   - MPF_DTMF_DETECTOR_BOTH: detect digits in both bands if supported by
 *     stream. When out-of-band digit arrives, in-band detection is turned off.
 * @param pool        Memory pool to allocate DTMF detector from.
 * @return The object or NULL on error.
 * @see mpf_dtmf_detector_create
 */
func DtmfDetectorCreateEx(stream *AudioStream, band DtmfDetectorBand) *DtmfDetector {
	var (
		flgBand = band
	)
	if stream.TXDescriptor == nil {
		flgBand &= ^MPF_DTMF_DETECTOR_INBAND
	}
	/*
		Event descriptor is not important actually
		if (!stream->tx_event_descriptor) flg_band &= ~MPF_DTMF_DETECTOR_OUTBAND;
	*/
	if flgBand <= 0 {
		return nil
	}
	det := new(DtmfDetector)
	det.Band = flgBand

	if det.Band&MPF_DTMF_DETECTOR_INBAND > 0 {
		for i := 0; i < DTMF_FREQUENCIES; i++ {
			det.energies[i].Coef = 2 * math.Cos(2*math.Pi*DtmfFreqs[i]/float64(stream.TXDescriptor.SamplingRate))
			det.energies[i].S1 = 0
			det.energies[i].S2 = 0
		}
		det.NSamples = 0
		det.WSamples = GOERTZEL_SAMPLES_8K * int64(stream.TXDescriptor.SamplingRate/8000)
		det.last1 = 0
		det.last2 = 0
		det.curr = 0
	}

	return det
}

/**
 * Create MPF DTMF detector (simple). Calls mpf_dtmf_detector_create_ex
 * with band = MPF_DTMF_DETECTOR_BOTH if out-of-band supported by the stream,
 * MPF_DTMF_DETECTOR_INBAND otherwise.
 * @param stream      A stream to get digits from.
 * @param pool        Memory pool to allocate DTMF detector from.
 * @return The object or NULL on error.
 * @see mpf_dtmf_detector_create_ex
 */
func DtmfDetectorCreate(stream *AudioStream) *DtmfDetector {
	var band DtmfDetectorBand = MPF_DTMF_DETECTOR_INBAND
	if stream.TXEventDescriptor != nil {
		band = MPF_DTMF_DETECTOR_BOTH
	}
	return DtmfDetectorCreateEx(stream, band)
}

/**
 * Get DTMF digit from buffer of digits detected so far and remove it.
 * @param detector  The detector.
 * @return DTMF character [0-9*#A-D] or NUL if the buffer is empty.
 */
func (detector *DtmfDetector) DtmfDetectorDigitGet() byte {
	var digit byte
	detector.mutex.Lock()
	defer detector.mutex.Unlock()
	digit = detector.buf[0]
	if digit > 0 {
		copy(detector.buf[:], detector.buf[1:])
		detector.Digits--
	}
	return digit
}

/**
 * Retrieve how many digits was lost due to full buffer.
 * @param detector  The detector.
 * @return Number of lost digits.
 */
func (detector *DtmfDetector) DtmfDetectorDigitsLost() int64 {
	return detector.LostDigits
}

/**
 * Empty the buffer and reset detection states.
 * @param detector  The detector.
 */
func (detector *DtmfDetector) DtmfDetectorReset() {
	detector.mutex.Lock()
	defer detector.mutex.Unlock()
	detector.buf[0] = 0
	detector.LostDigits = 0
	detector.Digits = 0
	detector.curr = 0
	detector.last1 = 0
	detector.last2 = 0
	detector.NSamples = 0
	detector.TotalEnergy = 0
}

func (detector *DtmfDetector) DtmfDetectorAddDigit(digit byte) {
	if digit <= 0 {
		return
	}
	detector.mutex.Lock()
	defer detector.mutex.Unlock()
	if detector.Digits < MPF_DTMFDET_BUFFER_LEN {
		detector.buf[detector.Digits] = digit
		detector.Digits++
		detector.buf[detector.Digits] = 0
	} else {
		detector.LostDigits++
	}
}

func (detector *DtmfDetector) GoertzelSample(sample int16) {
	for i := 0; i < DTMF_FREQUENCIES; i++ {
		s := detector.energies[i].S1
		detector.energies[i].S1 = detector.energies[i].S2
		detector.energies[i].S2 = float64(sample) + detector.energies[i].Coef*detector.energies[i].S1 - s
	}

	detector.TotalEnergy += float64(sample * sample)
}

func (detector *DtmfDetector) GoertzelEnergiesDigit() {

}

/**
 * Detect DTMF digits in the frame.
 * @param detector  The detector.
 * @param frame     Frame object passed in stream_write().
 */
func (detector *DtmfDetector) DtmfDetectorGetFrame(frame *Frame) {

}

/**
 * Free all resources associated with the detector.
 * @param detector  The detector.
 */
func DtmfDetectorDestroy(detector *DtmfDetector) error {
	return nil
}
