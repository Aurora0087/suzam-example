package hash

import (
	"suzam-example/db"
	"suzam-example/mytypes"
	"suzam-example/suzam/constellation"
)


func GenerateHashes(peaks []constellation.Peak) []db.Fingerprint {
	fingerprints := []db.Fingerprint{}

	const (
		TargetZoneTimeStart = 5
		TargetZoneTimeEnd   = 30
		MaxTargetsPerAnchor = 10 
	)

	for i := 0; i < len(peaks); i++ {
		anchor := peaks[i]
		foundForThisAnchor := 0

		for j := i + 1; j < len(peaks); j++ {
			target := peaks[j]

			timeDiff := target.Frame - anchor.Frame

			if timeDiff < TargetZoneTimeStart {
				continue
			}
			if timeDiff > TargetZoneTimeEnd {
				break
			}

			// Anchor Freq (9 bits), Target Freq (9 bits), Time Diff (14 bits)
			hash := uint32(anchor.Bin&0x1FF)<<23 |
				uint32(target.Bin&0x1FF)<<14 |
				uint32(timeDiff&0x3FFF)

			fingerprints = append(fingerprints, db.Fingerprint{
				Hash:       hash,
				AnchorTime: anchor.Frame,
			})

			foundForThisAnchor++
			if foundForThisAnchor >= MaxTargetsPerAnchor {
				break
			}
		}
	}
	return fingerprints
}

func GenerateHashesForClip(peaks []constellation.Peak) []mytypes.ClipFingerprint {
	fingerprints := []mytypes.ClipFingerprint{}

	const (
		TargetZoneTimeStart = 5
		TargetZoneTimeEnd   = 30
	)

	for i := 0; i < len(peaks); i++ {
		anchor := peaks[i]

		for j := i + 1; j < len(peaks); j++ {
			target := peaks[j]

			timeDiff := target.Frame - anchor.Frame

			if timeDiff < TargetZoneTimeStart {
				continue
			}
			if timeDiff > TargetZoneTimeEnd {
				break
			}

			// Anchor Freq (9 bits), Target Freq (9 bits), Time Diff (14 bits)
			hash := uint32(anchor.Bin&0x1FF)<<23 |
				uint32(target.Bin&0x1FF)<<14 |
				uint32(timeDiff&0x3FFF)

			fingerprints = append(fingerprints, mytypes.ClipFingerprint{
				Hash:       hash,
				AnchorTime: anchor.Frame,
				Value: anchor.Value,
			})
		}
	}
	return fingerprints
}