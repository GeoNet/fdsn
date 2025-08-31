package ms

import (
	"encoding/binary"
	"fmt"
)

func getNibble(word []byte, index int) uint8 {
	b := word[index/4] //Which byte we want from within the word (4 bytes per word)
	var res uint8      //value
	i := index % 4     //which nibble we want from within the byte (4 nibbles per byte)
	//nolint:gosec
	res = b & (0x3 << uint8((3-i)*2)) //0x3=00000011 create and apply the correct mask e.g. i=1 mask=00110000
	//nolint:gosec
	res = res >> uint8((3-i)*2) //shift the masked value fully to the right
	return res
}

// value must be 0, 1, 2 or 3, the nibble must not have been previously set
func writeNibble(word []byte, index int, value uint8) {
	b := word[index/4]
	i := index % 4
	//nolint:gosec
	b = b ^ (value << uint8((3-i)*2)) //set the bits
	word[index/4] = b
}

/*
Takes v: an integer where only the first numbits bits are used to represent the number and returns an int32
*/
func uintVarToInt32(v uint32, numbits uint8) int32 {
	neg := (v & (0x1 << (numbits - 1))) != 0 //check positive/negative
	if neg {                                 //2s complement
		v = v ^ ((1 << (numbits)) - 1) //flip all the bits
		v = v + 1                      //add 1 - positive nbit number
		v = -v                         //get the negative - this gives us a proper negative int32
	}
	return int32(v) //nolint:gosec
}

func int32ToUintVar(i int32, numbits uint8) uint32 {
	neg := i < 0
	if neg {
		i = -i //get the positive - this gives us a positive n bit int
		i = i - 1
		i = i ^ ((1 << (numbits)) - 1) //flip all the bits
	}
	return uint32(i) //nolint:gosec
}

func applyDifferencesFromWord(w []byte, numdiffs int, diffbits uint32, d []int32) []int32 {
	mask := (1 << diffbits) - 1
	wint := binary.BigEndian.Uint32(w)

	for i := numdiffs - 1; i >= 0; i-- {
		//nolint:gosec
		intn := wint & uint32(mask<<(uint32(i)*diffbits)) //apply a mask over the correct bits
		//nolint:gosec
		intn = intn >> (uint32(i) * diffbits) //shift the masked value fully to the right
		//nolint:gosec
		diff := uintVarToInt32(intn, uint8(diffbits)) //convert diffbits bit int to int32

		d = append(d, d[len(d)-1]+diff)
	}

	return d
}

func decodeSteim(version int, raw []byte, wordOrder, frameCount uint8, expectedSamples uint16) ([]int32, error) {
	d := make([]int32, 0, expectedSamples)

	if wordOrder == 0 {
		return d, fmt.Errorf("steim%v: no support for little endian", version)
	}

	//Word 1 and 2 contain x0 and xn: the uncompressed initial and final quantities (word 0 contains nibs)
	frame0 := raw[0:64]
	start := int32(binary.BigEndian.Uint32(frame0[4:8])) //nolint:gosec
	end := int32(binary.BigEndian.Uint32(frame0[8:12]))  //nolint:gosec

	d = append(d, start)

	for f := 0; f < int(frameCount); f++ {
		//Each frame is 64bytes
		frameBytes := raw[64*f : 64*(f+1)]

		//The first word (w0) of the frames contains 16 'nibbles' (2bit codes)
		w0 := frameBytes[:4]

		//Each nibble describes the encoding on a 32bit word of the frame
		//0 is w0 so we can skip that
		for w := 1; w < 16; w++ {
			nib := getNibble(w0, w) //get relevant nib

			if nib == 0 { //A 0 nib represents a non-data or header word
				continue
			}

			wb := frameBytes[w*4 : (w+1)*4] //Get word w

			//dnib is the second part of the encoding specifier and is the first nib of the word (not used in steim1)
			var dnib uint8
			if version == 2 && nib != 1 { //nib == 1 indicates no dnib (steim1 8bit encoding
				dnib = getNibble(wb, 0)
			}

			skipFirstDiff := 0
			if w == 3 && f == 0 { //libmseed seems to skip the first diff? TODO: Understand
				skipFirstDiff = 1
			}

			if version == 2 {
				switch nib {
				case 1: //4 8bit differences
					d = applyDifferencesFromWord(wb, 4-skipFirstDiff, 8, d)
				case 2:
					switch dnib {
					case 0:
						return d, fmt.Errorf("steim%v: nib 10 dnib 00 is an illegal configuration @ frame %v word %v", version, f, w)
					case 1:
						d = applyDifferencesFromWord(wb, 1-skipFirstDiff, 30, d)
					case 2:
						d = applyDifferencesFromWord(wb, 2-skipFirstDiff, 15, d)
					case 3:
						d = applyDifferencesFromWord(wb, 3-skipFirstDiff, 10, d)
					}
				case 3:
					switch dnib {
					case 0: //5 6bit differences
						d = applyDifferencesFromWord(wb, 5-skipFirstDiff, 6, d)
					case 1: //6 5bit differences
						d = applyDifferencesFromWord(wb, 6-skipFirstDiff, 5, d)
					case 2: //7 4bit differences
						d = applyDifferencesFromWord(wb, 7-skipFirstDiff, 4, d)
					case 3:
						return d, fmt.Errorf("steim%v: nib 11 dnib 11 is an illegal configuration @ frame %v word %v", version, f, w)
					}
				}
			} else if version == 1 {
				switch nib {
				case 1: //4 8bit differences
					d = applyDifferencesFromWord(wb, 4-skipFirstDiff, 8, d)
				case 2: //2 16bit differences
					d = applyDifferencesFromWord(wb, 2-skipFirstDiff, 16, d)
				case 3: //1 32bit difference
					d = applyDifferencesFromWord(wb, 1-skipFirstDiff, 32, d)
				}
			}
		}
	}

	if d[len(d)-1] != end {
		return d, fmt.Errorf("steim%v: final decompressed value did not equal reverse integration value got: %v expected %v", version, d[len(d)-1], end)
	}

	return d, nil
}
