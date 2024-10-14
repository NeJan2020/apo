package model

import "fmt"

type ErrWithMessage struct {
	Err  error
	Code string
}

func (e ErrWithMessage) Error() string {
	return e.Err.Error()
}

func NewErrWithMessage(err error, code string) ErrWithMessage {
	return ErrWithMessage{
		Err:  err,
		Code: code,
	}
}

type ErrAlertImpactMissingTag struct {
	TagGroups []TagGroup
	Event     *AlertEvent
}

type TagGroup []string

func (e ErrAlertImpactMissingTag) Error() string {
	return fmt.Sprintf("Unable to find any of the following label group %s", e.TagGroups)
}

func (e ErrAlertImpactMissingTag) CheckedTagGroups() string {
	return fmt.Sprintf("%s", e.TagGroups)
}

func (e *ErrAlertImpactMissingTag) AddCheckedGroup(err ErrAlertImpactMissingTag) {
	e.TagGroups = append(e.TagGroups, err.TagGroups...)
}
