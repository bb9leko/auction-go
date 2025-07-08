# 1. Build Imagem Docker
```docker compose up --build```

# 2. Log Container App
```docker-compose logs -f app```

# 3. Curl para criar Leilão
``` 
curl -X POST http://localhost:8080/auction \                                                     
  -H "Content-Type: application/json" \
  -d '{
    "product_name": "Test 77",
    "category": "Electronics", 
    "description": "Test description",
    "condition": 0
  }'

```

# 4. Acessar container MongoDB
```docker exec -it mongodb mongosh "mongodb://admin:admin@localhost:27017/auctions?authSource=admin"```

```use auctions```

```db.auctions.find().pretty()```
