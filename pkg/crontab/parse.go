package crontab

import (
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
)

type timeRange struct {
	min  uint
	max  uint
	name map[string]uint
}

var timeRanges = map[int]timeRange{
	0: timeRange{0, 59, nil},
	1: timeRange{0, 59, nil},
	2: timeRange{0, 23, nil},
	3: timeRange{1, 31, nil},
	4: timeRange{1, 12, map[string]uint{
		"jan": 1,
		"feb": 2,
		"mar": 3,
		"apr": 4,
		"may": 5,
		"jun": 6,
		"jul": 7,
		"aug": 8,
		"sep": 9,
		"oct": 10,
		"nov": 11,
		"dec": 12,
	}},
	5: timeRange{1, 7, map[string]uint{
		"sun": 1,
		"mon": 2,
		"tue": 3,
		"wed": 4,
		"thu": 5,
		"fri": 6,
		"sat": 7,
	}},
}

var (
	invalid       = errors.New("cron time expr invalid")
	commaUseError = errors.New("comma use error")
	slashUseError = errors.New("slash use error")
	lineUseError  = errors.New("line use error")
	outOfRange    = errors.New("out of range")
	matchEmpty    = errors.New("match empty")
)

type Parse struct {
	expr     string
	isParse  bool
	second   []uint
	minute   []uint
	hour     []uint
	day      []uint
	month    []uint
	week     []uint
	dayExpr  string
	weekExpr string
}

func NewParse(expr string) *Parse {
	return &Parse{
		expr:    expr,
		isParse: false,
	}
}

type nextExecTime struct {
	ct       time.Time
	isChange bool
	year     uint
	month    uint
	day      uint
	hour     uint
	minute   uint
	second   uint
}

func (p *Parse) NextExecTime(t time.Time) (time.Time, error) {
	parseErr := p.parse()
	if parseErr != nil {
		return t, parseErr
	}
	nt := &nextExecTime{
		ct:       t,
		isChange: false,
		year:     uint(t.Year()),
		month:    uint(t.Month()),
	}
	if !p.isExistUintSlice(p.month, nt.month) {
		nt.isChange = true
		p.setNextMonth(nt)
	}
	nt.day = uint(t.Day())
	err := p.setNextDay(nt)
	if err != nil {
		return t, err
	}
	if nt.isChange {
		nt.hour = p.hour[0]
	} else {
		nt.hour = uint(t.Hour())
		if !p.isExistUintSlice(p.hour, nt.hour) {
			nt.isChange = true
			err = p.setNextHour(nt)
			if err != nil {
				return t, err
			}
		}
	}
	if nt.isChange {
		nt.minute = p.minute[0]
	} else {
		nt.minute = uint(t.Minute())
		if !p.isExistUintSlice(p.minute, nt.minute) {
			nt.isChange = true
			err = p.setNextMinute(nt)
			if err != nil {
				return t, err
			}
		}
	}
	if nt.isChange {
		nt.second = p.second[0]
	} else {
		nt.second = uint(t.Second())
		if !p.isExistUintSlice(p.second, nt.second) {
			nt.isChange = true
			err = p.setNextSecond(nt)
			if err != nil {
				return t, err
			}
		}
	}
	if nt.isChange == false {
		nt.isChange = true
		err = p.setNextSecond(nt)
		if err != nil {
			return t, err
		}
	}
	lt := p.getLocation("CST", 8*3600)
	return time.Date(int(nt.year), time.Month(int(nt.month)), int(nt.day), int(nt.hour), int(nt.minute), int(nt.second), 0, lt), nil
}

func (p *Parse) setNextMonth(nt *nextExecTime) {
	us, ok := p.getNearestUintSlice(p.month, nt.month, true)
	if ok {
		nt.year += 1
	}
	nt.month = us
}

func (p *Parse) setNextDay(nt *nextExecTime) error {
	var err error
	if len(p.day) > 0 {
		if nt.isChange {
			nt.day = p.day[0]
			err = p.handleMonthDayOver(nt)
		} else if !p.isExistUintSlice(p.day, nt.day) {
			nt.isChange = true
			us, ok := p.getNearestUintSlice(p.day, nt.day, true)
			if ok {
				p.setNextMonth(nt)
			}
			nt.day = us
			err = p.handleMonthDayOver(nt)
		}
		return err
	}
	if len(p.week) > 0 {
		days := p.getDayByWeek(nt.year, nt.month, p.week)
		if nt.isChange {
			nt.day = days[0]
		} else if !p.isExistUintSlice(days, nt.day) {
			nt.isChange = true
			bi := p.getBiggerItemInUintSlice(days, nt.day)
			if bi == 0 {
				p.setNextMonth(nt)
				days = p.getDayByWeek(nt.year, nt.month, p.week)
				bi = days[0]
			}
			nt.day = bi
		}
		return err
	}
	if p.dayExpr == "L" {
		dayCount := p.getDayCountByMonth(nt.year, nt.month)
		if nt.isChange {
			nt.day = dayCount
		} else if nt.day < dayCount {
			nt.isChange = true
			p.setNextMonth(nt)
			nt.day = dayCount
		}
		return err
	}
	if strings.Contains(p.dayExpr, "W") {
		ps := strings.TrimRight(p.dayExpr, "W")
		var lw uint
		var d int
		if ps == "L" {
			lw = p.getNearestWeekday(nt.year, nt.month, 31)
		} else {
			d, err = strconv.Atoi(ps)
			if err != nil {
				return invalid
			}
			lw = p.getNearestWeekday(nt.year, nt.month, uint(d))
		}
		if nt.isChange {
			nt.day = lw
		} else if lw != nt.day {
			nt.isChange = true
			if lw < nt.day {
				p.setNextMonth(nt)
				if ps == "L" {
					lw = p.getNearestWeekday(nt.year, nt.month, 31)
				} else {
					d, err = strconv.Atoi(ps)
					if err != nil {
						return invalid
					}
					lw = p.getNearestWeekday(nt.year, nt.month, uint(d))
				}
			}
			nt.day = lw
		}
		return err
	}
	if strings.Contains(p.weekExpr, "L") {
		var ld int
		ld, err = strconv.Atoi(strings.TrimRight(p.weekExpr, "L"))
		if err != nil {
			return invalid
		}
		var days []uint
		var md uint
		wd := uint(ld) - 1
		days = p.getDayByWeek(nt.year, nt.month, []uint{wd})
		md = days[len(days)-1]
		if nt.isChange {
			nt.day = md
		} else if md != nt.day {
			nt.isChange = true
			if md < nt.day {
				p.setNextMonth(nt)
				days = p.getDayByWeek(nt.year, nt.month, []uint{wd})
				md = days[len(days)-1]
			}
			nt.day = md
		}
		return err
	}
	if strings.Contains(p.weekExpr, "#") {
		s := strings.Split(p.weekExpr, "#")
		if len(s) != 2 {
			return invalid
		}
		var n, w int
		w, err = strconv.Atoi(s[0])
		if err != nil {
			return invalid
		}
		n, err = strconv.Atoi(s[1])
		if err != nil {
			return invalid
		}
		var days []uint
		var d uint
		wd := uint(w) - 1
		days = p.getDayByWeek(nt.year, nt.month, []uint{wd})
		if n > len(days) {
			return invalid
		}
		d = days[uint(n)-1]
		if nt.isChange {
			nt.day = d
		} else if d != nt.day {
			nt.isChange = true
			if d < nt.day {
				p.setNextMonth(nt)
				days = p.getDayByWeek(nt.year, nt.month, []uint{wd})
				d = days[uint(n)-1]
			}
			nt.day = d
		}
		return err
	}
	return invalid
}

func (p *Parse) setNextHour(nt *nextExecTime) error {
	var err error
	nh, ok := p.getNearestUintSlice(p.hour, nt.hour, true)
	if ok {
		if len(p.day) > 0 {
			var us uint
			us, ok = p.getNearestUintSlice(p.day, nt.day, true)
			if ok {
				p.setNextMonth(nt)
			}
			nt.day = us
			err = p.handleMonthDayOver(nt)
			if err != nil {
				return err
			}
		} else if len(p.week) > 0 {
			var days []uint
			bi := p.getBiggerItemInUintSlice(days, nt.day)
			if bi == 0 {
				p.setNextMonth(nt)
				days = p.getDayByWeek(nt.year, nt.month, p.week)
				bi = days[0]
			}
			nt.day = bi
		} else if p.dayExpr == "L" {
			p.setNextMonth(nt)
			dayCount := p.getDayCountByMonth(nt.year, nt.month)
			nt.day = dayCount
		} else if strings.Contains(p.dayExpr, "W") {
			p.setNextMonth(nt)
			ps := strings.TrimRight(p.dayExpr, "W")
			var lw uint
			if ps == "L" {
				lw = p.getNearestWeekday(nt.year, nt.month, 31)
			} else {
				var d int
				d, err = strconv.Atoi(ps)
				if err != nil {
					return invalid
				}
				lw = p.getNearestWeekday(nt.year, nt.month, uint(d))
			}
			nt.day = lw
		} else if strings.Contains(p.weekExpr, "L") {
			var ld int
			ld, err = strconv.Atoi(strings.TrimRight(p.weekExpr, "L"))
			if err != nil {
				return invalid
			}
			wd := uint(ld) - 1
			p.setNextMonth(nt)
			days := p.getDayByWeek(nt.year, nt.month, []uint{wd})
			nt.day = days[len(days)-1]
		} else if strings.Contains(p.weekExpr, "#") {
			s := strings.Split(p.weekExpr, "#")
			if len(s) != 2 {
				return invalid
			}
			var n, w int
			w, err = strconv.Atoi(s[0])
			if err != nil {
				return invalid
			}
			n, err = strconv.Atoi(s[1])
			if err != nil {
				return invalid
			}
			wd := uint(w) - 1
			p.setNextMonth(nt)
			days := p.getDayByWeek(nt.year, nt.month, []uint{wd})
			if n > len(days) {
				return invalid
			}
			nt.day = days[uint(n)-1]
		} else {
			return invalid
		}
	}
	nt.hour = nh
	return nil
}

func (p *Parse) setNextMinute(nt *nextExecTime) error {
	var err error
	nm, ok := p.getNearestUintSlice(p.minute, nt.minute, true)
	if ok {
		err = p.setNextHour(nt)
		if err != nil {
			return err
		}
	}
	nt.minute = nm
	return nil
}

func (p *Parse) setNextSecond(nt *nextExecTime) error {
	var err error
	ns, ok := p.getNearestUintSlice(p.second, nt.second, true)
	if ok {
		err = p.setNextMinute(nt)
		if err != nil {
			return err
		}
	}
	nt.second = ns
	return nil
}

func (p *Parse) handleMonthDayOver(nt *nextExecTime) error {
	if nt.day <= 28 {
		return nil
	}
	var err error
	var dt string
	var isMatch bool
	layout := "2006-01-02"
	for i := 1; i <= 12; i++ {
		url := "%d-%.2d-%.2d"
		dt = fmt.Sprintf(url, nt.year, nt.month, nt.day)
		_, err = time.Parse(layout, dt)
		if err == nil {
			isMatch = true
			break
		}
		p.setNextMonth(nt)
		nt.day = p.day[0]
	}
	if isMatch {
		return nil
	} else {
		return matchEmpty
	}
}

func (p *Parse) parse() error {
	if p.isParse {
		return nil
	}
	exprSlices := strings.Split(p.expr, " ")
	if len(exprSlices) != 6 {
		return invalid
	}
	p.dayExpr = exprSlices[3]
	p.weekExpr = exprSlices[5]
	var err error
	for k, exprSlice := range exprSlices {
		if k < 3 {
			err = p.parseTime(exprSlice, k)
		} else {
			err = p.parseDate(exprSlice, k)
		}
		if err != nil {
			return err
		}
	}
	p.isParse = true
	return nil
}

func (p *Parse) parseTime(expr string, k int) error {
	t, err := p.parseCommonExpr(expr, timeRanges[k])
	if err != nil {
		return err
	}
	switch k {
	case 0:
		p.second = t
	case 1:
		p.minute = t
	case 2:
		p.hour = t
	}
	return nil
}

func (p *Parse) parseDate(expr string, k int) error {
	t, err := p.parseCommonExpr(expr, timeRanges[k])
	if err != nil {
		return err
	}
	if t != nil {
		switch k {
		case 3:
			p.day = t
		case 4:
			p.month = t
		case 5:
			for i, _ := range t {
				p.week[i] = t[i] - 1
			}
		}
	}
	return nil
}

func (p *Parse) parseCommonExpr(expr string, tg timeRange) ([]uint, error) {
	var t []uint
	if expr == "*" {
		for i := tg.min; i <= tg.max; i++ {
			t = append(t, i)
		}
		return t, nil
	}
	var s int
	s, err := strconv.Atoi(expr)
	if err == nil {
		t = append(t, uint(s))
		return t, nil
	}
	var us uint
	var ok bool
	slices := strings.Split(expr, ",")
	lh := uint(len(slices))
	if lh > tg.max {
		return nil, outOfRange
	} else if lh > 1 {
		for _, slice := range slices {
			s, err = strconv.Atoi(slice)
			if err == nil {
				us = uint(s)
				if us < tg.min && us > tg.max {
					return nil, outOfRange
				}
			} else {
				if tg.name == nil {
					return nil, commaUseError
				}
				us, ok = tg.name[strings.ToLower(slice)]
				if !ok {
					return nil, commaUseError
				}
			}
			t = append(t, us)
		}
		sort.Slice(t, func(i, j int) bool {
			return t[i] < t[j]
		})
		return t, nil
	}
	slices = strings.Split(expr, "/")
	lh = uint(len(slices))
	if lh > 2 {
		return nil, slashUseError
	} else if lh == 2 {
		var start, step uint
		if slices[0] == "*" {
			start = 0
		} else {
			s, err = strconv.Atoi(slices[0])
			if err == nil {
				start = uint(s)
				if start <= tg.min && start >= tg.max {
					return nil, outOfRange
				}
			} else {
				if tg.name == nil {
					return nil, slashUseError
				}
				start, ok = tg.name[strings.ToLower(slices[0])]
				if !ok {
					return nil, slashUseError
				}
			}
		}
		s, err = strconv.Atoi(slices[1])
		if err != nil {
			return nil, slashUseError
		}
		step = uint(s)
		if step > tg.max || step > tg.max-start {
			return nil, slashUseError
		}
		for i := start; i <= tg.max; i += step {
			t = append(t, i)
		}
		return t, nil
	}
	slices = strings.Split(expr, "-")
	lh = uint(len(slices))
	if lh > 2 {
		return nil, lineUseError
	} else if lh == 2 {
		var start, end uint
		s, err = strconv.Atoi(slices[0])
		if err == nil {
			start = uint(s)
			if start < tg.min && start > tg.max {
				return nil, outOfRange
			}
		} else {
			if tg.name == nil {
				return nil, lineUseError
			}
			start, ok = tg.name[strings.ToLower(slices[0])]
			if !ok {
				return nil, lineUseError
			}
		}
		s, err = strconv.Atoi(slices[1])
		if err == nil {
			end = uint(s)
			if end < tg.min && end > tg.max {
				return nil, outOfRange
			}
		} else {
			if tg.name == nil {
				return nil, lineUseError
			}
			end, ok = tg.name[strings.ToLower(slices[0])]
			if !ok {
				return nil, lineUseError
			}
		}
		if start < end {
			for i := start; i <= end; i++ {
				t = append(t, i)
			}
		} else {
			var i uint
			for i = start; i <= tg.max; i++ {
				t = append(t, i)
			}
			for i = tg.min; i <= end; i++ {
				if start != end {
					t = append(t, i)
				}
			}
			sort.Slice(t, func(i, j int) bool {
				return t[i] < t[j]
			})
		}
		return t, nil
	}
	return t, nil
}

func (p *Parse) isExistUintSlice(s []uint, i uint) bool {
	for _, v := range s {
		if v == i {
			return true
		}
	}
	return false
}

func (p *Parse) getLocation(locationName string, locationOffset int) *time.Location {
	return time.FixedZone(locationName, locationOffset)
}

func (p *Parse) getNearestUintSlice(s []uint, i uint, d bool) (uint, bool) {
	if !d {
		ns := make([]uint, len(s))
		copy(ns, s)
		sort.Slice(ns, func(i, j int) bool {
			return s[i] > s[j]
		})
		s = ns
	}
	for _, v := range s {
		if (d && v > i) || (!d && v < i) {
			return v, false
		}
	}
	return s[0], true
}

func (p *Parse) getDayByWeek(year uint, month uint, weekdays []uint) []uint {
	var days = make([]uint, 0)
	dayCount := int(p.getDayCountByMonth(year, month))
	location := p.getLocation("CST", 8*3600)
	for i := 1; i <= dayCount; i++ {
		t := time.Date(int(year), time.Month(int(month)), i, 0, 0, 0, 0, location)
		if p.isExistUintSlice(weekdays, uint(t.Weekday())) {
			days = append(days, uint(i))
		}
	}
	return days
}

func (p *Parse) getDayCountByMonth(year, month uint) uint {
	monthDay := [12]uint{31, 28, 31, 30, 31, 30, 31, 31, 30, 31, 30, 31}
	monthDayLeapYear := [12]uint{31, 29, 31, 30, 31, 30, 31, 31, 30, 31, 30, 31}
	if p.isLeapYear(year) {
		return monthDayLeapYear[month-1]
	}
	return monthDay[month-1]
}

func (p *Parse) isLeapYear(year uint) bool {
	if year%100 != 0 && year%4 == 0 {
		return true
	}
	if year%100 == 0 && year%400 == 0 {
		return true
	}
	return false
}

func (p *Parse) getBiggerItemInUintSlice(s []uint, i uint) uint {
	for _, v := range s {
		if v > i {
			return v
		}
	}
	return 0
}

func (p *Parse) getNearestWeekday(year uint, month uint, day uint) uint {
	location := p.getLocation("CST", 8*3600)
	dayCount := p.getDayCountByMonth(year, month)
	if day == 31 {
		day = dayCount
	}
	var s = []int{0, 1, -1, -2, 2}
	var t time.Time
	var d int
	for _, v := range s {
		d = int(day)
		d += v
		if d > int(dayCount) || d < 1 {
			continue
		}
		t = time.Date(int(year), time.Month(int(month)), d, 0, 0, 0, 0, location)
		if t.Weekday() == time.Saturday || t.Weekday() == time.Sunday {
			continue
		}
		break
	}
	return uint(d)
}
