package gortcut

import "regexp"

_#StateName: =~"^[a-z]+$"

_#JobAction: close({
	show?: string
	goto:  _#StateName | "STAY"
})

_#Reg: string & regexp.Valid

_#CaseTerm: close({
	stdout?:   _#Reg
	stderr?:   _#Reg
	exitcode?: int
	success?:  bool
})

_#Case: [_#CaseTerm, ..._#CaseTerm]

_#Do: close({
	show?: string
	goto:  _#StateName | "STAY"
})

_#Match: {
	case!: _#Case
	do!:   _#Do
}

_#State: {
	name:  string
	title: string
	cmd: [string, ...string]
	autorun: *false | bool
	match: [..._#Match]
}

_#Job: close({
	title: string
	start: _#StateName
	states: {
		[ST=_#StateName]: close(_#State & {
			name: *ST | string
		})
	}
	_startState: states[start]
	if _startState == _|_ {
		start_not_in_states: [...][0]
	}
	{
		for st, state in states
		for i, case in state.match
		if case.do.goto != "STAY" && states[case.do.goto] == _|_ {
			("state_\(st)_case_\(i)_goto_\(case.do.goto)_illegal"): [...][0]
		}
	}
})

jobs: [..._#Job]
