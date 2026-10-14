# Leilão com fechamento automático

Sistema de leilões em Go (base: `devfullcycle/labs-auction-goexpert`) com **fechamento automático** de leilões via goroutine.

## Como funciona

Em `internal/infra/database/auction/create_auction.go`, ao criar um leilão o repositório:

1. Insere o leilão no MongoDB com status `Active` (`0`).
2. Dispara uma goroutine (`closeAuctionAt`) que espera até `timestamp + AUCTION_DURATION` usando um `time.Timer`, sem bloquear a requisição.
3. Quando o tempo expira, faz um `UpdateOne` alterando o status para `Completed` (`1`). O filtro inclui `status: Active`, então o update é idempotente.

A mesma duração (`GetAuctionDuration`) é usada pela rotina de lances, então a validação de lances e o fechamento automático usam sempre o mesmo prazo.

## Variáveis de ambiente

Configuradas em `cmd/auction/.env`:

| Variável | Descrição | Padrão |
|---|---|---|
| `AUCTION_DURATION` | Duração do leilão (formato `time.ParseDuration`: `30s`, `5m`, `1h`) | `5m` |
| `AUCTION_INTERVAL` | Usada como fallback se `AUCTION_DURATION` não estiver definida | — |
| `BATCH_INSERT_INTERVAL` | Intervalo de inserção em lote dos lances | `20s` |
| `MAX_BATCH_SIZE` | Tamanho máximo do lote de lances | `4` |
| `MONGODB_URL` | URL de conexão do MongoDB | — |
| `MONGODB_DB` | Nome do banco | — |

Para mudar a duração, edite `AUCTION_DURATION` em `cmd/auction/.env` e suba os containers novamente.

## Rodando com Docker

```bash
docker compose up --build
```

A API sobe em `http://localhost:8080`.

### Testando manualmente

```bash
curl -X POST localhost:8080/auction \
  -H 'Content-Type: application/json' \
  -d '{"product_name":"Celular","category":"Eletronicos","description":"Celular seminovo com caixa","condition":1}'

curl 'localhost:8080/auction?status=0'
```

Após `AUCTION_DURATION`, o leilão aparece com `"status": 1` (Completed):

```bash
curl 'localhost:8080/auction?status=1'
```

## Testes

O teste de fechamento automático precisa de um MongoDB. Suba apenas o banco e rode os testes:

```bash
docker compose up -d mongodb
go test ./internal/infra/database/auction/ -v
```

O teste `TestCreateAuction_ClosesAutomaticallyAfterDuration`:

1. Configura `AUCTION_DURATION=2s`.
2. Cria um leilão e verifica que o status é `Active`.
3. Aguarda 3 segundos.
4. Verifica que o status mudou para `Completed` sem intervenção manual.

Por padrão o teste conecta em `mongodb://admin:admin@localhost:27017/?authSource=admin`; para usar outro endereço, defina `MONGODB_TEST_URL`. Se o MongoDB não estiver acessível, o teste é ignorado (`skip`).
