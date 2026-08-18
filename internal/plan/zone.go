package plan

// MOWL computes program TSS from each interval's ScaleCoggan power zone (1-7),
// NOT from the FTP percentages — every zone carries a fixed intensity factor
// (measured live against the API). TSS = Σ durationₕ × IF(zone)² × 100.
//
// zoneIF[z] is the intensity factor MOWL uses for Coggan zone z.
var zoneIF = map[int]float64{
	1: 0.2775,
	2: 0.6575,
	3: 0.8300,
	4: 0.9800,
	5: 1.1300,
	6: 1.3575,
	7: 1.5025,
}

// CogganZone maps a percent-of-FTP intensity to a Coggan power zone (1-7),
// using the standard zone boundaries.
func CogganZone(ftpPct int) int {
	switch {
	case ftpPct <= 55:
		return 1
	case ftpPct <= 75:
		return 2
	case ftpPct <= 90:
		return 3
	case ftpPct <= 105:
		return 4
	case ftpPct <= 120:
		return 5
	case ftpPct <= 150:
		return 6
	default:
		return 7
	}
}

// ZoneIF returns MOWL's intensity factor for a Coggan zone (1-7). Out-of-range
// zones clamp to the nearest valid zone.
func ZoneIF(zone int) float64 {
	if zone < 1 {
		zone = 1
	}
	if zone > 7 {
		zone = 7
	}
	return zoneIF[zone]
}
