# Leilão (Auction) - Desafio Go Expert

Sistema de leilão com fechamento automático de leilões baseado em tempo configurável.

## Funcionalidade: Fechamento Automático de Leilões

O sistema fecha automaticamente leilões expirados usando uma **goroutine** de monitoramento que roda em background. A cada intervalo configurado (`AUCTION_CLOSE_CHECK_INTERVAL`), a rotina verifica no MongoDB quais leilões estão com status `Active` e cujo `timestamp` ultrapassou o tempo de duração configurado (`AUCTION_INTERVAL`), atualizando-os para `Completed`.

### Arquitetura da Solução

- **Arquivo principal da implementação:** `internal/infra/database/auction/create_auction.go`
- `NewAuctionRepository` inicia uma goroutine (`monitorAuctionExpiration`) que usa `time.Ticker`
- A goroutine é cancelável via `context.Context` para shutdown gracioso
- `closeExpiredAuctions` executa `UpdateMany` no MongoDB filtrando auctions ativas e expiradas
- A validação de leilão fechado na criação de bids já estava implementada em `internal/infra/database/bid/create_bid.go`

## Variáveis de Ambiente

| Variável | Descrição | Valor Padrão |
|---|---|---|
| `AUCTION_INTERVAL` | Duração de um leilão (tempo até expirar) | `5m` |
| `AUCTION_CLOSE_CHECK_INTERVAL` | Intervalo entre verificações de leilões expirados | `10s` |
| `BATCH_INSERT_INTERVAL` | Intervalo para flush do batch de bids | `3m` |
| `MAX_BATCH_SIZE` | Tamanho máximo do batch de bids | `5` |
| `MONGODB_URL` | URL de conexão com o MongoDB | — |
| `MONGODB_DB` | Nome do banco de dados MongoDB | — |

## Como Rodar o Projeto (Dev)

### Pré-requisitos

- Docker e Docker Compose instalados

### Subir a aplicação

```bash
docker-compose up --build
```

Isso irá:
1. Subir o MongoDB na porta `27017`
2. Compilar e iniciar a aplicação Go na porta `8080`
3. Iniciar a goroutine de monitoramento de leilões expirados

### Endpoints da API

| Método | Rota | Descrição |
|---|---|---|
| `POST` | `/auction` | Criar um novo leilão |
| `GET` | `/auction` | Listar leilões (filtro por status, categoria, nome) |
| `GET` | `/auction/:auctionId` | Buscar leilão por ID |
| `GET` | `/auction/winner/:auctionId` | Buscar lance vencedor de um leilão |
| `POST` | `/bid` | Criar um lance |
| `GET` | `/bid/:auctionId` | Listar lances de um leilão |
| `GET` | `/user/:userId` | Buscar usuário por ID |

### Exemplo de Uso

```bash
# Criar um leilão
curl -X POST http://localhost:8080/auction \
  -H "Content-Type: application/json" \
  -d '{"product_name": "iPhone 15", "category": "electronics", "description": "iPhone 15 Pro Max 256GB novo na caixa", "condition": 0}'

# Listar leilões ativos
curl http://localhost:8080/auction?status=0

# Após AUCTION_INTERVAL (20s no .env), o leilão será fechado automaticamente
# Verificar status atualizado:
curl http://localhost:8080/auction/{auctionId}
```

## Como Rodar os Testes

Os testes utilizam `mtest` (mock do MongoDB driver) e **não requerem** um MongoDB rodando.

```bash
# Rodar apenas testes do auction
go test ./internal/infra/database/auction/ -v
```

