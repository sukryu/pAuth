# 📚 pAuth: The Flexible and Scalable Authentication Platform

![pAuth Logo](<img src="public/pAuth.webp">)

**pAuth**는 다양한 요구사항에 따라 커스터마이징이 가능한 인증 및 권한 관리 플랫폼입니다. 기존의 Firebase Auth, Keycloak, Supabase 같은 도구가 제공하는 기능을 바탕으로, 대규모 플랫폼 수준의 사용자 인증 및 인가 서비스를 제공합니다. 

단순한 소규모 서비스부터 복잡한 엔터프라이즈 환경까지 대응할 수 있도록 설계되었으며, **사용자 정의 설정을 손쉽게 공유**할 수 있는 생태계를 지향합니다. 우리의 목표는 단순히 작동하는 서비스가 아니라, 유연성과 확장성을 통해 **모든 개발자가 자신의 플랫폼에 꼭 맞는 인증 솔루션을 구축할 수 있는 도구**를 제공하는 것입니다.

---

## 🌟 **프로젝트 개요**
- **목표**
  - 다양한 환경에 대응 가능한 고성능 인증 및 권한 관리 시스템 구축
  - **커스터마이징이 가능한 설정**을 통해 플랫폼을 자유롭게 확장
  - 대규모 시스템에서도 안정적으로 작동하는 엔터프라이즈 수준의 서비스 제공
  - 사용자 정의 설정 및 정책을 **다른 개발자와 공유할 수 있는 기능** 추가

---

## 🛠️ **현재까지의 작업 현황**

### **1. 아키텍처 설계**
- ✅ **구성 요소**
  - 사용자(User), 역할(Role), 역할 바인딩(RoleBinding)을 관리하는 Store 설계
  - Dynamic Store를 활용한 데이터베이스 테이블 생성 및 관리
- ✅ **DatabaseConfig**
  - SQLite 초기 설정 및 PostgreSQL 확장 가능하도록 설계
- ✅ **Dynamic Store**
  - 동적 테이블 생성, 데이터 삽입, 업데이트, 삭제 구현
  - 스키마 검증 및 인덱스 관리 기능 추가

---

### **2. Store 구현**
- ✅ **UserStore**
  - 사용자 CRUD 및 고유 필드 처리 구현
  - 테스트 커버리지: 67.2%
- ✅ **RoleStore**
  - 역할 CRUD 및 정책 규칙 관리 구현
  - 테스트 커버리지: 81.8%
- ✅ **RoleBindingStore**
  - 역할 바인딩 CRUD 및 Subject 관리 구현
  - 테스트 커버리지: 80.0%

---

### **3. Store Factory**
- ✅ Store Factory 구현
  - 여러 데이터베이스 매니저를 관리하며 Store와 Dynamic Store를 연동

---

### **4. 테스트**
- ✅ 각 Store의 단위 테스트 작성
- ◯ 통합 테스트: Factory 및 서버 실행 로직 포함

---

### **5. 패키지 구조**
- ✅ `internal/store`: Dynamic Store 및 각 Store(User, Role, RoleBinding) 구현
- ✅ `internal/config`: 설정 파일 로딩 및 DatabaseConfig 관리
- ✅ `internal/db`: SQLC 기반 데이터베이스 레이어
- ✅ `pkg/apis/auth/v1alpha1`: API 모델 정의
- ✅ `pkg/controllers`: 인증 및 권한 컨트롤러 구현
- ◯ `cmd/server`: 메인 서버 실행 로직

---

## 🔧 **남은 작업**

### **기능 구현**
- ◯ **OAuth 2.0 및 OpenID Connect 지원**
  - Authorization Code, Client Credentials, Implicit Grant 등 다양한 흐름 지원
- ◯ **사용자 정의 설정 공유**
  - 커스텀 인증 정책 및 권한 설정을 JSON/YAML 형태로 내보내기 및 가져오기 지원
- ◯ **대규모 환경 최적화**
  - 수백만 사용자를 대상으로 한 대규모 환경 지원 테스트

---

### **테스트**
- ◯ **통합 테스트**
  - SQLite와 Redis를 포함한 테스트 환경 설정 및 검증

---

### **Main 서버 로직**
- ◯ `cmd/server/main.go` 수정 및 기본 서버 실행 기능 추가

---

### **로그 및 모니터링**
- ◯ **ELK Stack** 또는 **Prometheus/Grafana**를 활용한 로깅 및 모니터링

---

### **PostgreSQL 지원**
- ◯ SQLite에서 PostgreSQL로 전환 가능한 데이터베이스 구성 테스트

---

### **보안 강화**
- ◯ JWT 서명 검증 및 만료 처리
- ◯ TOTP 기반 MFA 추가
- ◯ API 요청 제한 기능 (Rate Limiting)

---

### **문서화**
- ◯ API 문서 작성 (OpenAPI/Swagger)
- ◯ 아키텍처 설계 문서 작성

---

### **CI/CD**
- ◯ GitHub Actions 기반 자동 테스트 및 배포 파이프라인 설정

---

## 🚀 **다음 단계**
1. 통합 테스트 작성 및 현재 Store와 Factory 기능 검증
2. `cmd/server` 수정 및 기본 서버 실행 기능 추가
3. CI/CD 구성 및 테스트 자동화
4. 사용자 정의 정책 공유 기능 추가

---

💡 **Contributions are welcome!**  
궁금한 점이 있다면 언제든지 문의해주세요. 😊

![Cute Illustration](https://via.placeholder.com/150x100?text=Happy+Coding!)
