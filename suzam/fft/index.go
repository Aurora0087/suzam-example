package fft

import (
	"math"
	"math/cmplx"
)


func FFT(a []complex128) []complex128 {
	n := len(a)
	if n <= 1 {
		return a
	}

	// spliting into even and odd
	even := make([]complex128, n/2)
	odd := make([]complex128, n/2)

	for i := 0; i < n/2; i++ {
		even[i] = a[2*i]
		odd[i] = a[2*i+1]
	}

	// d.c.
	evenRes := FFT(even)
	oddRes := FFT(odd)

	// combine using twiddle factors
	result := make([]complex128, n)
	for k := 0; k < n/2; k++ {
		//twiddle factor: e^(-2*PI*i*k / N)
		angle := -2 * math.Pi * float64(k) / float64(n)
		twiddle := cmplx.Exp(complex(0, angle)) * oddRes[k]

		result[k] = evenRes[k] + twiddle
		result[k+n/2] = evenRes[k] - twiddle
	}
	return result
}



func ProcessWindowedFrame(frame []float32) []float64 {
	n := len(frame)

	input := make([]complex128, n)
	for i, val := range frame {
		input[i] = complex(float64(val), 0)
	}

	fftResult := FFT(input)

	magnitudes := make([]float64, n/2)
	for i := 0; i < n/2; i++ {
		//magnitude = sqrt(real^2 + imag^2)
		magnitudes[i] = cmplx.Abs(fftResult[i])

	}

	return magnitudes
}