# Favori Ürün Görselleri Tasarımı

Tarih: 2026-09-24

## Amaç

ZeytinERP ürünlerine internet adresi üzerinden görsel eklenmesini, görselin yalnızca bir kez backend tarafından indirilip sunucuda kalıcı saklanmasını ve görseli bulunan favori ürünlerin Hızlı Satış ekranında resimli kartlar olarak gösterilmesini sağlamak.

## Kapsam ve kurallar

- Ürün ekleme ve düzenleme formunda opsiyonel bir **Görsel URL’si** alanı bulunur.
- Görsel URL’si verildiğinde tarayıcı görseli doğrudan kullanmaz. Backend görseli indirir, doğrular ve yerel dosya olarak saklar.
- Ürün kaydında yalnızca sunucudaki yerel yol tutulur: `/uploads/products/<üretilen-ad>.<uzantı>`.
- Görsel URL’si gönderilmeden yapılan ürün güncellemesi mevcut görseli korur.
- Yeni URL başarıyla indirildiğinde eski yerel görsel değiştirilir; başarısız indirme mevcut ürün ve görseli değiştirmez.
- Görseli olmayan ürün favoriye alınamaz.
- Favori ürün sayısı mevcut kuraldaki gibi en fazla 10’dur.
- Daha önce favori yapılmış fakat görseli olmayan ürünler migration sırasında favoriden çıkarılır.
- Hızlı Satış ekranı yalnızca aktif, favori ve sunucuda kayıtlı görsel yolu bulunan ürünleri gösterir.

## Yaklaşım

Görsel indirme işlemi ürünün tek **Kaydet** akışının parçasıdır. Ayrı bir manuel yükleme ekranı veya arka plan kuyruğu kullanılmaz. Bu yaklaşım kullanıcı için tek adımlıdır ve başarısız indirmeyi kaydetme anında görünür kılar.

Frontend, ürün oluşturma/güncelleme isteğine yalnızca kullanıcı yeni bir internet adresi girdiyse `image_source_url` alanını ekler. Backend resmi geçici dosyaya indirip doğruladıktan sonra ürün kaydını ve dosya değişimini tamamlar.

## Backend tasarımı

### API sözleşmesi

`POST /api/products` ve `PUT /api/products/:id` istekleri aşağıdaki opsiyonel alanı kabul eder:

```json
{
  "image_source_url": "https://ornek.com/urun.webp"
}
```

Ürün cevaplarındaki mevcut `image_url` alanı yerel sunucu yolunu döndürür:

```json
{
  "image_url": "/uploads/products/7f0c....webp"
}
```

Frontend `image_url` değerini API sunucusunun kök adresiyle birleştirerek kullanır.

### İndirme ve dosya saklama

- Varsayılan dizin: `uploads/products`.
- Dizin uygulama başlarken veya ilk indirmede güvenli izinlerle oluşturulur.
- Dosya adı kullanıcı girdisinden üretilmez; rastgele/benzersiz bir kimlik kullanılır.
- İndirme geçici dosyaya yapılır ve doğrulama sonrasında atomik olarak hedef dizine taşınır.
- Desteklenen içerikler: JPEG, PNG ve WebP.
- SVG, HTML ve diğer içerikler reddedilir.
- Azami dosya boyutu 5 MiB, HTTP zaman aşımı 10 saniyedir.
- İçerik türü yalnızca HTTP başlığına göre değil, dosya imzası üzerinden de doğrulanır.
- HTTP 2xx dışındaki cevaplar reddedilir.

### Ağ güvenliği

Sunucunun URL üzerinden kendi iç ağına erişmesini önlemek için:

- Yalnızca `http` ve `https` şemaları kabul edilir.
- Kullanıcı bilgisi içeren URL’ler reddedilir.
- Localhost, loopback, private, link-local, multicast ve belirtilmemiş IP aralıkları reddedilir.
- Alan adı çözümlendikten sonra elde edilen bütün IP’ler doğrulanır.
- Her yönlendirme hedefi aynı kurallarla yeniden doğrulanır.
- DNS çözümlemesi ile bağlantı hedefinin farklılaşmasına karşı özel transport/dial kontrolü uygulanır.

### Tutarlılık ve temizleme

- Yeni ürün oluşturulurken görsel önce doğrulanıp geçici olarak hazırlanır. Veritabanı kaydı başarısız olursa geçici dosya silinir.
- Ürün güncellenirken yeni indirme başarısızsa eski veritabanı kaydı ve eski görsel korunur.
- Yeni kayıt başarılı olduktan sonra eski ürün görseli, yalnızca ürün yükleme dizini içindeyse silinir.
- Ürün silindiğinde ilişkili yerel ürün görseli de güvenli biçimde kaldırılır.
- `Update` işlemi favori durumu ve görsel gibi istekte bulunmayan alanları sıfırlamaz.

### Favori doğrulaması

`PUT /api/products/:id/favorite` isteğinde `isFavorite: true` olduğunda:

1. Ürünün aktif olduğu doğrulanır.
2. `image_url` değerinin ürün yükleme dizinindeki gerçek bir dosyaya karşılık geldiği doğrulanır.
3. Görsel yoksa HTTP 409 ve Türkçe mesaj döner: **“Favoriye eklemek için önce ürün görseli ekleyin.”**
4. Ardından mevcut 10 favori sınırı uygulanır.

### Dosya sunumu

Public route eklenir:

`GET /uploads/products/*filepath`

Route yalnızca ürün yükleme dizini altındaki dosyaları sunar; dizin dışına çıkmaya çalışan yollar engellenir. Ürün görselleri kimlik doğrulaması gerektirmez çünkü POS kartlarında ve ileride mobil vitrinde kullanılacaktır.

## Veritabanı migration’ı

Yeni sütun gerekmez; mevcut `products.image_url` kullanılır.

Yeni migration, görselsiz mevcut favorileri temizler:

```sql
UPDATE products
SET is_bestseller = false,
    bestseller_order = 0
WHERE is_bestseller = true
  AND COALESCE(TRIM(image_url), '') = '';
```

Migration geri alınırken eski favori seçiminin güvenilir biçimde yeniden kurulması mümkün olmadığı için down migration veri üretmez; şema değişikliği yoktur.

## Frontend tasarımı

### Ürün modeli ve servis

- `Product` tipine `imageUrl?: string` eklenir.
- Backend `image_url` alanı `imageUrl` alanına eşlenir.
- Ürün oluşturma/güncelleme tiplerine `imageSourceUrl?: string` eklenir.
- Kullanıcı URL girdiyse API isteğinde `image_source_url` gönderilir.
- Görsel URL’si boş bırakıldığında güncelleme isteği bu alanı göndermez ve mevcut görsel korunur.
- API kök adresinden sunulan yerel görsel yollarını tam URL’ye çeviren tek bir yardımcı fonksiyon kullanılır.

### Ürün formu

- “Görsel URL’si” alanı ürün bilgilerinin yanında bulunur.
- Düzenlemede mevcut yerel görsel önizlemesi gösterilir.
- Yeni internet adresi yazıldığında kayıt sırasında backend indirmesi beklenir.
- Başarısız URL, desteklenmeyen dosya veya boyut aşımı backend’in Türkçe hata mesajıyla form üzerinde gösterilir.
- Başarılı kayıt sonrasında normal kapanış akışı devam eder.
- İnternet adresi doğrudan `img src` olarak önizlenmez; tarayıcının dış kaynağa sürekli erişmesi ve izleme riski engellenir.

### Favori yönetimi

- Görseli bulunmayan üründe favori düğmesi devre dışı gösterilir veya tıklamada açıklayıcı uyarı verir.
- Asıl kural backend’de uygulanır; frontend kontrolü yalnızca kullanıcı deneyimini iyileştirir.
- Görsel kaydedildikten sonra ürün listesi yenilenir ve favoriye alma işlemi kullanılabilir olur.

### Hızlı Satış kartları

- Mevcut `QuickProductsGrid` 5 sütun × 2 satır düzenini korur.
- Her kartta sunucudan gelen ürün görseli, ürün adı ve satış fiyatı gösterilir.
- Görsel alanı karta sığacak şekilde `object-contain` kullanır; ambalaj fotoğrafları kırpılmaz.
- Kart tıklaması mevcut şekilde ürünü sepete ekler.
- Hızlı Satış bağlamı ürün/favori değişikliğinden sonra güncel listeyi yükler.
- Bozuk veya sonradan silinmiş dosya istemci tarafından favori kuralını aşmak için kullanılmaz; backend doğrulaması kaynak gerçektir.

## Hata mesajları

- Boş/bozuk URL: “Geçerli bir görsel adresi girin.”
- Erişilemeyen kaynak: “Görsel adresine ulaşılamadı.”
- Desteklenmeyen içerik: “Yalnızca JPG, PNG veya WebP görselleri kullanılabilir.”
- Boyut aşımı: “Görsel en fazla 5 MB olabilir.”
- Görselsiz favori: “Favoriye eklemek için önce ürün görseli ekleyin.”

## Test stratejisi

### Backend

- Başarılı JPEG/PNG/WebP indirme ve yerel dosya kaydı.
- 5 MiB sınırı, içerik türü, bozuk veri ve zaman aşımı.
- Localhost/private IP ve yasak yönlendirme hedeflerinin reddi.
- Başarısız create/update işleminde geçici dosyanın temizlenmesi.
- Görsel değişiminde eski dosyanın kaldırılması ve başarısız değişimde korunması.
- Görseli olmayan ürünün favoriye alınamaması.
- Görselli üründe favori sırası ve 10 ürün sınırının korunması.
- Public ürün görseli route’u ve path traversal engeli.
- Ürün güncellemesinin mevcut favori ve görsel alanlarını koruması.

### Frontend

- `image_url` alanının `imageUrl` olarak eşlenmesi.
- Create/update isteklerinin yalnızca yeni URL girildiğinde `image_source_url` göndermesi.
- Ürün formunda mevcut görsel önizlemesi ve backend hata mesajı.
- Görselsiz üründe favori işleminin engellenmesi.
- Favori kartında yerel sunucu görseli, ürün adı ve fiyatın gösterilmesi.
- Kart tıklamasının ürünü sepete eklemeye devam etmesi.

## Dağıtım ve işletim

- `uploads/products` dizini Git tarafından yönetilmez ve dağıtımlarda silinmez.
- Sunucu güncellemesinde `git clean -fd` çalıştırılmaz.
- Backend servis kullanıcısının dizine yazma izni doğrulanır.
- Önce backend migration ve servis, ardından frontend build dağıtılır.
- Dağıtım sonrasında bir ürün URL ile kaydedilir; dış kaynak kapatılsa bile yerel görselin açıldığı doğrulanır.
- Yedekleme kapsamına PostgreSQL ile birlikte `uploads/products` dizini de dahil edilir.
