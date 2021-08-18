# payment-system
simple payments API  

to run:
```shell
docker-compose up
```
then run cmd/payments/main.go (requires ENV)
```yml
PG_DSN: postgresql://user:user_pw@localhost:5433/payments?sslmode=disable
```