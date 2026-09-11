# qme50ir

Sonicake Matribox (QME-50)용 절차적 캐비닛 IR 생성기 + `.prst` 프리셋 빌더 (Go, 테스트 포함).

## 빌드

```bash
go build -o qme50ir ./cmd/qme50ir
go test ./...
```

## 사용

```bash
qme50ir list                                  # 캐비닛 카탈로그 (4x12 V30, 1x12 Blues, ...)
qme50ir gen -out irs -n 3 -type "4x12 V30"    # IR WAV 생성 (seed로 변형)
qme50ir preset -in prsts.prst -out new.prst \
  -template 5 -name "V30 Proced" -ir 168820742 # 프리셋 클론 → 유저 IR 슬롯 연결
```

## 구조

| 경로 | 내용 |
|---|---|
| `ir/` | IR 합성 (감쇠 노이즈 + RBJ 피킹 바이쿼드 + 얼리 리플렉션), WAV 인코딩은 `go-audio/wav` |
| `prst/` | QME-50 `.prst` XML 번역 파서/라이터 (동적 요소명 `ppIRInfoN`/`ppEXP1_N`·`params_N` 속성 보존) |
| `cmd/qme50ir/` | CLI (`list`, `gen`, `preset`) |

## 알고리즘 출처

합성 방식은 컨볼루션 리버브용 시뮬레이션 IR 생성기(예:
[adelespinasse/reverbGen](https://github.com/adelespinasse/reverbGen))와 동일 계열 —
캐비닛 스펙트럼 타깃으로 셰이핑한 지수 감쇠 노이즈 테일 + 스파스 얼리 리플렉션.
WAV IO는 Go 생태계 표준인 [go-audio/wav](https://github.com/go-audio/wav).

## 포맷 리서치

`.prst` 포맷 분석 노트: `docs/prst-format-research.md`
