# Template CRUD Multitenant - Documentação Completa

## 📋 Índice

1. [Visão Geral](#visão-geral)
2. [Arquitetura](#arquitetura)
3. [Multitenancy](#multitenancy)
4. [Estrutura do Projeto](#estrutura-do-projeto)
5. [Requisitos e Instalação](#requisitos-e-instalação)
6. [Fluxo de Requisições](#fluxo-de-requisições)
7. [Padrões Adotados](#padrões-adotados)
8. [Guia de Extensão](#guia-de-extensão)
9. [Exemplos Práticos](#exemplos-práticos)
10. [Glossário](#glossário)
11. [Diagramas](#diagramas)

---

## 🎯 Visão Geral

### O que é este template?

Este é um **template de CRUD Multitenant em Golang** que implementa uma arquitetura limpa e escalável para aplicações SaaS que precisam servir múltiplos clientes (tenants) de forma isolada e segura.

### Objetivo Principal

Fornecer uma base sólida e reutilizável para desenvolvimento de aplicações multitenant, permitindo que desenvolvedores:
- Criem rapidamente novos módulos seguindo padrões estabelecidos
- Garantam isolamento completo entre tenants
- Implementem autenticação e autorização baseada em roles
- Mantenham código limpo, testável e escalável

### Problema que Resolve

- **Duplicação de código**: Elimina a necessidade de reescrever lógica de multitenancy em cada projeto
- **Isolamento de dados**: Garante que cada tenant acesse apenas seus dados
- **Autenticação complexa**: Implementa sistema robusto de autenticação com tokens e roles
- **Arquitetura inconsistente**: Estabelece padrões claros de organização de código

### Como Usar

Este template deve ser usado como **ponto de partida** para novos projetos multitenant. Desenvolvedores devem:
1. Clonar o repositório
2. Configurar variáveis de ambiente
3. Executar migrations
4. Criar novos domínios seguindo o padrão existente (tenant/user)

### Cenários de Uso

- **Sistemas SaaS**: Onde cada cliente precisa de dados isolados
- **Marketplaces**: Com vendedores independentes
- **Plataformas educacionais**: Com múltiplas instituições
- **ERPs multi-empresa**: Gerenciando várias empresas

---

## 🏛️ Arquitetura

### Clean Architecture

Este projeto implementa **Clean Architecture** (Arquitetura Limpa), um padrão que separa responsabilidades em camadas concêntricas, onde camadas internas não conhecem camadas externas.

### Camadas da Arquitetura

```
┌─────────────────────────────────────────┐
│         Interface Layer (HTTP)          │  ← Controllers, DTOs, Routes
├─────────────────────────────────────────┤
│         Application Layer               │  ← Use Cases, Auth
├─────────────────────────────────────────┤
│         Domain Layer                    │  ← Entities, Business Rules
├─────────────────────────────────────────┤
│         Infrastructure Layer            │  ← Database, JWT, External APIs
└─────────────────────────────────────────┘
```

**Detalhamento das Camadas:**

#### 1️⃣ Domain Layer (Domínio)
- **Localização**: `internal/iam/domain/`
- **Responsabilidade**: Contém as regras de negócio e entidades do sistema
- **Componentes**:
  - **Models**: Entidades de domínio (`Tenant`, `User`)
  - **Services**: Lógica de negócio pura
  - **Repositories (Interface)**: Contratos para acesso a dados
  - **Errors**: Erros específicos do domínio

#### 2️⃣ Application Layer (Aplicação)
- **Localização**: `internal/iam/application/`
- **Responsabilidade**: Orquestra os casos de uso
- **Componentes**:
  - **Auth**: Autenticação e geração de tokens
  - **Use Cases**: Coordenação entre domínios

#### 3️⃣ Interface Layer (Interface HTTP)
- **Localização**: `cmd/server/`, `internal/iam/domain/*/controller.go`
- **Responsabilidade**: Expor funcionalidades via HTTP
- **Componentes**:
  - **Controllers**: Recebem requests HTTP e retornam responses
  - **DTOs**: Data Transfer Objects (request/response)
  - **Routes**: Configuração de rotas e middlewares

#### 4️⃣ Infrastructure Layer (Infraestrutura)
- **Localização**: `internal/infra/`
- **Responsabilidade**: Implementações técnicas
- **Componentes**:
  - **Database**: Conexão com PostgreSQL, migrations
  - **JWT**: Geração e validação de tokens
  - **Mailer**: Envio de e-mails

### Fluxo de Responsabilidades

```
HTTP Request → Controller → Service → Repository → Database
                    ↓            ↓          ↓
                  DTOs    Business Rules  SQL Query
```

**Exemplo de fluxo completo:**

1. **Controller** recebe request HTTP com dados do tenant
2. **Validation** valida estrutura do request
3. **Service** aplica regras de negócio (ex: validar documento único)
4. **Repository** executa query no banco de dados
5. **Database** retorna dados
6. **Repository** converte para entidade de domínio
7. **Service** retorna entidade
8. **Controller** converte para DTO de response
9. **HTTP Response** retorna JSON ao cliente

### Separação de Responsabilidades

| Camada | Conhece | Não Conhece |
|--------|---------|-------------|
| **Domain** | Entidades, Regras | HTTP, Database, JWT |
| **Application** | Domain | HTTP, Database |
| **Interface** | Application, Domain | Database details |
| **Infrastructure** | - | Domain rules |

---

## 🏢 Multitenancy

### Modelo Adotado

Este projeto implementa **Shared Database with Discriminator Column** (Banco de Dados Compartilhado com Coluna Discriminadora).

**Características:**
- ✅ Todos os tenants compartilham o mesmo banco de dados
- ✅ Cada tabela possui coluna `tenant_uuid` para identificação
- ✅ Queries sempre filtram por `tenant_uuid`
- ✅ Isolamento garantido pela aplicação
- ✅ Econômico e escalável para médio porte

**Comparação com outros modelos:**

| Modelo | Prós | Contras | Quando usar |
|--------|------|---------|-------------|
| **Schema-per-tenant** | Isolamento forte | Complexo | Poucos tenants grandes |
| **Database-per-tenant** | Isolamento total | Muito complexo | Tenants enterprise |
| **Discriminator Column** ✅ | Simples, econômico | Risco de vazamento | Muitos tenants pequenos |

### Identificação do Tenant

O tenant é identificado através do **token JWT** na requisição:

```go
// User possui referência ao Tenant
type User struct {
    UUID       uuid.UUID  `gorm:"type:uuid;primary_key"`
    TenantUUID *uuid.UUID `gorm:"type:uuid;index"`  // ← Relacionamento
    // ...
    Tenant     Tenant     `gorm:"foreignKey:TenantUUID"`
}
```

**Fluxo de identificação:**

1. Cliente envia `Authorization: Bearer <token>`
2. Middleware extrai e valida o token
3. Middleware busca dados do usuário associado ao token
4. Middleware injeta `tenant_uuid` no contexto da requisição
5. Repository usa `tenant_uuid` para filtrar queries

### Isolamento de Dados

O isolamento é garantido em **três níveis**:

#### Nível 1: Middleware de Autenticação
```go
// middleware.go
func (mw *impl) SetContextAutorization() gin.HandlerFunc {
    // Valida token e injeta dados do usuário no contexto
    SetAuthenticatedUser(c, login)
}
```

#### Nível 2: Contexto da Requisição
```go
// Usuário autenticado com tenant associado
type LoginResponse struct {
    User        model.User
    AcessToken  model.AcessToken
}
```

#### Nível 3: Repository Layer
```go
// Sempre filtra por tenant_uuid nas queries
query.Where("tenant_uuid = ?", userTenantUUID)
```

### Criação de Tenant

**Endpoint:** `POST /api/tenant/create`

**Request:**
```json
{
  "name": "Empresa XYZ",
  "document": "12345678901234"
}
```

**Processo:**
1. SYSTEM_ADMIN cria novo tenant
2. Tenant recebe UUID único
3. Tenant é persistido no banco
4. Usuários podem ser associados ao tenant

### Autenticação Multitenant

**Login:** `POST /api/auth/login`

```json
{
  "email": "user@empresa.com",
  "password": "senha123"
}
```

**Resposta:**
```json
{
  "access_token": "eyJhbGc...",
  "user": {
    "uuid": "...",
    "tenant_uuid": "...",  // ← Identifica o tenant
    "role": "TENANT_USER"
  }
}
```

### Roles e Permissões

| Role | Pode Acessar | Escopo |
|------|--------------|--------|
| `SYSTEM_ADMIN` | Todos os recursos | Global (sem tenant) |
| `TENANT_ADMIN` | Recursos do tenant | Apenas seu tenant |
| `TENANT_USER` | Recursos limitados | Apenas seu tenant |

**Exemplo de proteção de rota:**
```go
routes.POST("/tenant/create", 
    mw.SetContextAutorization(),
    mw.AuthorizeRole(model.RoleSystemAdmin),
    ctrl.Create,
)
```

### Garantindo Queries Seguras

✅ **Correto:**
```go
// Repository filtra automaticamente por tenant
db.Where("tenant_uuid = ?", tenantUUID).Find(&users)
```

❌ **Incorreto:**
```go
// NUNCA fazer query global sem filtro de tenant
db.Find(&users) // ← VAZAMENTO DE DADOS!
```

---

