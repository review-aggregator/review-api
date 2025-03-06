package consts

type PlatformType string

const (
	PlatformAll        PlatformType = "all"
	PlatformTrustpilot PlatformType = "trustpilot"
	PlatformAmazon     PlatformType = "amazon"
)

type TimePeriodType string

const (
	TimePeriodThisWeek  TimePeriodType = "this_week"
	TimePeriodLastWeek  TimePeriodType = "last_week"
	TimePeriodThisMonth TimePeriodType = "this_month"
	TimePeriodLastMonth TimePeriodType = "last_month"
	TimePeriodAllTime   TimePeriodType = "all_time"
)

var TimePeriods = []TimePeriodType{
	// TimePeriodThisWeek,
	// TimePeriodLastWeek,
	// TimePeriodThisMonth,
	// TimePeriodLastMonth,
	TimePeriodAllTime,
}
