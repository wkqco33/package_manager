package cmd

import (
	"fmt"
	"os"

	"github.com/wkqco33/wcli/rich"
)

// confirmDestructiveAction은 테스트에서 입력과 TTY 판정을 주입할 수 있는 확인 함수입니다.
var confirmDestructiveAction = defaultConfirmDestructiveAction

func requireDestructiveConfirmation(action string, bypass bool) error {
	if bypass {
		return nil
	}
	if noInputMode {
		return fmt.Errorf("%s 작업은 확인이 필요합니다. 비대화형 환경에서는 --yes 또는 --force를 사용하세요", action)
	}
	return confirmDestructiveAction(action)
}

func defaultConfirmDestructiveAction(action string) error {
	if !isTerminal(os.Stdin) {
		return fmt.Errorf("%s 작업은 TTY 대화형 확인이 필요합니다. 비대화형 환경에서는 --yes 또는 --force를 사용하세요", action)
	}
	confirmed, err := rich.Confirm(fmt.Sprintf("%s 작업을 계속하시겠습니까?", action), false)
	if err != nil {
		return fmt.Errorf("%s 확인 입력을 읽을 수 없습니다: %w", action, err)
	}
	if !confirmed {
		return fmt.Errorf("%s 작업이 취소되었습니다", action)
	}
	return nil
}
