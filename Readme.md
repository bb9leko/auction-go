# 1. Pare qualquer MongoDB rodando
```docker stop $(docker ps -q --filter "ancestor=mongo") 2>/dev/null || true```

# 2. Inicie MongoDB limpo
```docker run -d --name mongodb-test -p 27017:27017 mongo:latest --noauth```

# 3. Execute os testes
```sleep 5```
```go test ./internal/infra/database/auction -v```

# 4. Limpe
```docker stop mongodb-test && docker rm mongodb-test```