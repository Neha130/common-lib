package timeRangeLib

import (
	"fmt"
	"github.com/robfig/cron/v3"
	"time"
)

func (tr TimeRange) GetTimeRangeWindow(targetTime time.Time) (nextWindowEdge time.Time, isTimeBetween bool, err error) {
	err = tr.ValidateTimeRange()
	if err != nil {
		return nextWindowEdge, false, err
	}

	if tr.Frequency == Fixed {
		nextWindowEdge, isTimeBetween = getWindowForFixedTime(targetTime, tr)
		return nextWindowEdge, isTimeBetween, nil
	}

	windowStart, windowEnd, err := tr.getWindowForTargetTime(targetTime)
	if err != nil {
		return nextWindowEdge, isTimeBetween, err
	}
	if isTimeInBetween(targetTime, windowStart, windowEnd) {
		return windowEnd, true, nil
	}
	return windowStart, false, nil
}

func (tr TimeRange) getWindowForTargetTime(targetTime time.Time) (time.Time, time.Time, error) {
	duration, cronExp := tr.getDurationAndCronExp(targetTime)
	parser := cron.NewParser(CRON)
	schedule, err := parser.Parse(cronExp)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("error parsing cron expression %s %v", cronExp, err)
	}

	windowStart, windowEnd := tr.getWindowStartAndEndTime(targetTime, duration, schedule)
	return windowStart, windowEnd, nil
}

func (tr TimeRange) getDurationAndCronExp(targetTime time.Time) (time.Duration, string) {
	lastDayOfMonth := tr.calculateLastDayOfMonthForOverlappingWindow(targetTime)
	duration := tr.getDuration(lastDayOfMonth)
	cronExp := tr.getCron(lastDayOfMonth)
	return duration, cronExp
}

func (tr TimeRange) calculateLastDayOfMonthForOverlappingWindow(targetTime time.Time) int {
	month, year := tr.getMonthAndYearForPreviousWindow(targetTime)
	return getLastDayOfMonth(year, month)
}

func (tr TimeRange) getWindowStartAndEndTime(targetTime time.Time, duration time.Duration, schedule cron.Schedule) (time.Time, time.Time) {
	var windowEnd time.Time

	timeMinusDuration := tr.currentTimeMinusWindowDuration(targetTime, duration)
	windowStart := schedule.Next(timeMinusDuration)
	windowEnd = windowStart.Add(duration)

	if !tr.TimeFrom.IsZero() && windowStart.Before(tr.TimeFrom) {
		windowStart = tr.TimeFrom
	}
	if !tr.TimeTo.IsZero() && windowEnd.After(tr.TimeTo) {
		windowEnd = tr.TimeTo
	}
	return windowStart, windowEnd
}

func (tr TimeRange) currentTimeMinusWindowDuration(targetTime time.Time, duration time.Duration) time.Time {
	prevDuration := duration
	if tr.isMonthOverlapping() && !tr.isInsideOverLap(targetTime) {
		prevDuration = tr.getAdjustedDuration(targetTime, duration, prevDuration)
	}
	return targetTime.Add(-1 * prevDuration)
}

func (tr TimeRange) getAdjustedDuration(targetTime time.Time, duration time.Duration, prevDuration time.Duration) time.Duration {
	//adjusting duration when duration for consecutive windows is different
	currentMonth := targetTime.Month()
	currentYear := targetTime.Year()
	previousMonth, previousYear := getPreviousMonthAndYear(currentMonth, currentYear)
	diff := getLastDayOfMonth(currentYear, currentMonth) - getLastDayOfMonth(previousYear, previousMonth)
	prevDuration = duration - time.Duration(diff)*time.Hour*24
	return prevDuration
}

// this will determine if the relevant year and month for the last window happens
// in the same month or previous month
func (tr TimeRange) getMonthAndYearForPreviousWindow(targetTime time.Time) (time.Month, int) {
	month := targetTime.Month()
	year := targetTime.Year()

	if tr.isMonthOverlapping() && tr.isInsideOverLap(targetTime) {
		month, year = getPreviousMonthAndYear(month, year)
	}
	return month, year
}

func (tr TimeRange) isInsideOverLap(targetTime time.Time) bool {
	// for an overlapping window if the current time is on the latter part of the overlap then
	// we use the last month for calculation.
	day := targetTime.Day()
	if day < 1 {
		return false
	}
	return day < tr.DayTo || (day == tr.DayTo && tr.isToHourMinuteBeforeWindowEnd(targetTime))
}

func isTimeInBetween(timeCurrent, periodStart, periodEnd time.Time) bool {
	return (timeCurrent.After(periodStart) && timeCurrent.Before(periodEnd)) || timeCurrent.Equal(periodStart)
}

func getWindowForFixedTime(targetTime time.Time, timeRange TimeRange) (time.Time, bool) {
	var windowStartOrEnd time.Time
	if targetTime.After(timeRange.TimeTo) {
		return windowStartOrEnd, false
	} else if targetTime.Before(timeRange.TimeFrom) {
		return timeRange.TimeFrom, false
		//} else if targetTime.Before(timeRange.TimeTo) && targetTime.After(timeRange.TimeFrom) {
	} else if isTimeInBetween(targetTime, timeRange.TimeFrom, timeRange.TimeTo) {
		return timeRange.TimeTo, true
	}
	return windowStartOrEnd, false
}
