
# 3-3 商品一覧を取得する

商品の取得

```
$ curl \
  -X POST \
  --url 'http://localhost:9001/items' \
  -d 'name=jacket' \
  -d 'category=fashion'
```

登録された商品一覧

```
$ curl -X GET 'http://127.0.0.1:9001/items'
```

画像の登録

```
$ curl \
  -X POST \
  --url 'http://localhost:9001/items' \
  -F 'name=jacket' \
  -F 'category=fashion' \
  -F 'image=@test.jpg'
```

商品の詳細を返す

```
$ curl -X GET 'http://127.0.0.1:9001/items/1' 
```
