# Contributing

버그 수정, 문서 개선, 테스트 추가를 환영합니다.

## 시작하기

1. 저장소를 fork하고 작업 브랜치를 생성합니다.
2. 변경사항을 작고 명확한 커밋으로 작성합니다.
3. 아래 검증을 실행합니다.

```bash
task test
task test-race
task lint
gofmt -l .
```

`gofmt -l .`이 아무 파일도 출력하지 않아야 합니다.

## Pull Request

Pull Request에는 다음 내용을 포함해 주세요.

- 변경 목적과 사용자에게 미치는 영향
- 테스트 방법과 결과
- 호환성 또는 보안 관련 고려사항

기능 변경에는 가능한 한 회귀 테스트를 추가해 주세요. 보안 취약점은 공개 issue가 아니라 [Security Policy](SECURITY.md)에 따라 비공개로 신고해 주세요.
