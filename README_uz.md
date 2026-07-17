# HULK - HTTP Unbearable Load King

Serverni yuklab sinash vositasi. Raw TCP ulanishlar orqali nishonga katta hajmdagi HTTP so'rovlarini yuboradi.

## Ishlatish

```
./hulk -site http://target.com
```

To'xtatish uchun `Ctrl+C`.

## Kalitlar

| Flag | Standart | Tavsifi |
|------|---------|-------------|
| `-site` | `http://localhost` | Nishon URL |
| `-safe` | false | Server 500 qaytganda to'xtash |
| `-data` | "" | POST body (POST rejimiga o'tadi) |
| `-agents` | "" | Maxsus User-Agent ro'yxati (har qatordan bitta) |
| `-header` | | Qo'shimcha header (bir necha marta ishlatsa bo'ladi: `-header "X-Foo: bar"`) |
| `-version` | false | Versiyani chiqarish va chiqish |

## Muhit o'zgaruvchilari

- `HULKMAXPROCS` — bir vaqtda ishlaydigan goroutinlar soni (standart: 1023)

## Docker

```
docker build -t hulk -f docker/Dockerfile .
docker run -it hulk -site http://target.com
```

## Benchmark: grafov/hulk bilan solishtirish

| Nishon | Bu versiya (raw TCP) | grafov/hulk (net/http) | Tezlik farqi |
|--------|----------------------|----------------------|---------|
| Localhost (HTTP) | ~189,000 req/s | ~13,100 req/s | **14x** |
| Localhost (sekin server) | ~8,850 req/s | ~200 req/s | **44x** |
| Masofaviy sayt (HTTP/HTTPS) | ~4,800-7,500 req/s | ~1,200 req/s | **4-6x** |
| Masofaviy sayt (faqat HTTPS) | ~280-420 req/s | 0 (ulana olmadi) | — |

### Nega buncha tez?

Eski versiya (`net/http`) har bir so'rov uchun:
1. Yangi `http.Client` yaratadi (TLS handshake overhead)
2. Javobni o'qib `.Body.Close()` qiladi
3. Barcha goroutinlar bitta `chan` orqali natija qaytaradi — **bottleneck**

Bu versiya (raw TCP):
1. Har bir goroutine **bitta TCP ulanish** ochadi va keep-alive orqali ketma-ket so'rov yuboradi — TLS handshake faqat 1 marta
2. Javobni o'qimaydi — **zero-copy**
3. Pre-spawned goroutinlar, `sync/atomic` hisoblagichlar — **kanalsiz, bottlenecksiz**
4. Har bir goroutin o'z PRNG'iga ega (`rand.New(rand.NewSource(...))`) — **mutex contention yo'q**
5. To'g'ridan-to'g'ri `net.Dialer` + `net.TCPConn.Write()` — hech qanday pooling, wrapping, interface overheadi yo'q

---

Asos: [grafov/hulk](https://github.com/grafov/hulk) (Go port) va [Barry Shteiman's original HULK](http://www.sectorix.com/2012/05/17/hulk-web-server-dos-tool/) (Python).
