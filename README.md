# market-erp-backend

Go, Gin ve PostgreSQL ile yazılmış sade market cari takip ERP backend projesi.

Ana odak ürün/stok değil; firma borç takibi, personel maaş/avans takibi, günlük kasa, gelir-gider ve dashboard özetleridir. Ürün/stok endpointleri ayrı modül olarak korunur.

## Özellikler

- Gin REST API
- PostgreSQL + GORM
- `.env` üzerinden `DATABASE_URL`
- JWT auth ve bcrypt şifre hashleme
- CORS açık
- Standart JSON response yapısı
- Decimal/numeric para alanları
- Bakiye değerleri hareketlerden hesaplanır, tablolarda ayrıca tutulmaz

Başarılı response:

```json
{
  "success": true,
  "data": {}
}
```

Hata response:

```json
{
  "success": false,
  "error": "name is required"
}
```

## Kurulum

Ortam dosyasını oluştur:

```bash
cp .env.example .env
```

`.env` içindeki PostgreSQL bağlantısını düzenle:

```env
APP_ENV=development
PORT=8081
DATABASE_URL=postgres://postgres:postgres@localhost:5432/market_erp?sslmode=disable
JWT_SECRET=change_this_secret
CORS_ALLOWED_ORIGINS=http://zeytinerp.herevemarket.com,https://zeytinerp.herevemarket.com,http://localhost:5173
```

Production ortamında `JWT_SECRET` boş bırakılamaz. Development ortamında boşsa geçici development secret kullanılır, fakat gerçek kullanımda `.env` içine güçlü bir değer yazılmalıdır.
Production ortamında `CORS_ALLOWED_ORIGINS` içinde `*` kullanılmamalıdır; backend bunu reddeder.

Bağımlılıkları indir:

```bash
go mod tidy
```

Migration çalıştır:

```bash
set -a
source .env
set +a
psql "$DATABASE_URL" -f migrations/001_init.up.sql
psql "$DATABASE_URL" -f migrations/002_financial_debts.up.sql
psql "$DATABASE_URL" -f migrations/003_financial_installments_alerts.up.sql
psql "$DATABASE_URL" -f migrations/004_users_auth.up.sql
psql "$DATABASE_URL" -f migrations/005_supplier_transactions_current_account.up.sql
psql "$DATABASE_URL" -f migrations/006_supplier_transaction_files.up.sql
psql "$DATABASE_URL" -f migrations/007_bank_wallet.up.sql
psql "$DATABASE_URL" -f migrations/008_wallet.up.sql
psql "$DATABASE_URL" -f migrations/009_finance_center_indexes.up.sql
psql "$DATABASE_URL" -f migrations/010_supplier_transaction_currencies.up.sql
psql "$DATABASE_URL" -f migrations/011_supplier_transaction_payment_return_files.up.sql
psql "$DATABASE_URL" -f migrations/012_supplier_cash_payment_index.up.sql
psql "$DATABASE_URL" -f migrations/013_supplier_visit_days.up.sql
```

Uygulama açılırken migration dosyalarını otomatik de çalıştırır.

Uygulamayı başlat:

```bash
go run cmd/server/main.go
```

Default admin:

- Kullanıcı adı: `admin`
- Şifre: `123456`
- Rol: `admin`

Üretimde ilk girişten sonra default admin şifresi değiştirilmelidir.

Token olmadan korumalı API testi:

```bash
curl http://localhost:8081/api/suppliers
```

Beklenen sonuç HTTP `401`:

```json
{
  "success": false,
  "error": "authorization token is required"
}
```

Token al:

```bash
curl -X POST http://localhost:8081/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"123456"}'
```

Token ile istek:

```bash
curl http://localhost:8081/api/suppliers \
  -H "Authorization: Bearer TOKEN"
```

## Endpointler

Health:

- `GET /health`

Auth:

- `POST /api/auth/login`
- `GET /api/auth/me`

`/health` ve `/api/auth/login` açıktır. Diğer `/api` endpointleri `Authorization: Bearer TOKEN` header’ı ister.

Dashboard:

- `GET /api/dashboard`
- `GET /api/dashboard/monthly?year=2026&month=5`

Raporlar:

- `GET /api/reports`
- `GET /api/reports/monthly?year=2026&month=5`

Dashboard finans alanları:

- `total_financial_debt`
- `total_bank_balance`
- `wallet_balance`
- `monthly_financial_due`
- `overdue_financial_count`
- `upcoming_financial_due_7_days`
- `upcoming_financial_due_30_days`
- `nearest_financial_installments`

Firmalar:

- `POST /api/suppliers`
- `GET /api/suppliers`
- `GET /api/suppliers?search=cola`
- `GET /api/suppliers/:id`
- `PUT /api/suppliers/:id`
- `DELETE /api/suppliers/:id`
- `GET /api/suppliers/:id/balance`
- `GET /api/suppliers-balances`
- `GET /api/suppliers-currency-totals`
- `GET /api/firms/today-visits`
- `GET /api/firms/visits?day=monday`

Firma oluşturma ve güncelleme isteklerinde `visit_days` alanı haftanın birden fazla
gününü alır. Değerler `monday`, `tuesday`, `wednesday`, `thursday`, `friday`,
`saturday`, `sunday` olarak saklanır; Türkçe gün adları da kabul edilip bu değerlere
dönüştürülür. Örnek: `"visit_days": ["monday", "thursday"]`.

Firma hareketleri:

- `POST /api/supplier-transactions`
- `GET /api/supplier-transactions`
- `GET /api/supplier-transactions?supplier_id=1`
- `GET /api/supplier-transactions?supplier_id=1&type=invoice`
- `GET /api/supplier-transactions/:id`
- `PUT /api/supplier-transactions/:id`
- `DELETE /api/supplier-transactions/:id`
- `POST /api/supplier-transactions/:id/files`
- `GET /api/supplier-transactions/:id/files`
- `DELETE /api/supplier-transaction-files/:file_id`
- `DELETE /api/invoices/:id`
- `GET /uploads/invoices/:file_name`

Kurlar:

- `GET /api/exchange-rates/latest?currency=USD`
- `GET /api/exchange-rates/latest?currency=EUR`

Personel:

- `POST /api/employees`
- `GET /api/employees`
- `GET /api/employees/:id`
- `PUT /api/employees/:id`
- `DELETE /api/employees/:id`
- `GET /api/employees/:id/balance`
- `GET /api/employees-balances`

Personel hareketleri:

- `POST /api/employee-transactions`
- `GET /api/employee-transactions`
- `GET /api/employee-transactions?employee_id=1`
- `DELETE /api/employee-transactions/:id`

Günlük kasa:

- `POST /api/daily-cash-reports`
- `GET /api/daily-cash-reports`
- `GET /api/daily-cash-reports/:id`
- `PUT /api/daily-cash-reports/:id`
- `DELETE /api/daily-cash-reports/:id`

Giderler:

- `POST /api/expenses`
- `GET /api/expenses`
- `GET /api/expenses?start_date=2026-05-01&end_date=2026-05-31`
- `GET /api/expenses/by-date?date=2026-07-01`
- `PUT /api/expenses/:id`
- `DELETE /api/expenses/:id`

`start_date` ve `end_date` filtreleri iki uç dahil olacak şekilde uygulanır. Liste cevabındaki `total_amount`, aynı tarih filtresindeki giderlerin toplamıdır.

`GET /api/expenses/by-date` yalnızca `YYYY-MM-DD` biçiminde zorunlu bir `date` parametresi alır. Seçilen günün tüm giderlerini sayfalama olmadan `expense_date DESC` sırasıyla döndürür; `total` ve `count` backend tarafından hesaplanır. Filtre, `expense_date >= gün başlangıcı AND expense_date < ertesi gün` aralığını kullanır.

Örnek cevap:

```json
{
  "success": true,
  "date": "2026-07-01",
  "total": "5425.50",
  "count": 8,
  "expenses": [
    {
      "id": 15,
      "expense_date": "2026-07-01T00:00:00Z",
      "category": "market_gideri",
      "amount": "1250",
      "payment_method": "Nakit",
      "note": "ABC Gıda"
    }
  ]
}
```

Para alanları, diğer finans endpointlerinde olduğu gibi hassasiyet kaybını önlemek için JSON string olarak döner. Mevcut `expenses.expense_date` kolonu `DATE` tipinde olduğundan saat bilgisi saklanmaz.

Gelirler:

- `POST /api/income-entries`
- `GET /api/income-entries`
- `GET /api/income-entries?start_date=2026-05-01&end_date=2026-05-31`
- `PUT /api/income-entries/:id`
- `DELETE /api/income-entries/:id`

`start_date` ve `end_date` filtreleri iki uç dahil olacak şekilde uygulanır. Liste cevabındaki `total_amount`, aynı tarih filtresindeki gelirlerin toplamıdır.

Finans borçları:

- `POST /api/financial-debts`
- `GET /api/financial-debts`
- `GET /api/financial-debts/:id`
- `PUT /api/financial-debts/:id`
- `DELETE /api/financial-debts/:id`
- `GET /api/financial-debts/:id/balance`
- `GET /api/financial-debts/:id/summary`

Finans taksitleri:

- `POST /api/financial-debts/:id/installments`
- `GET /api/financial-debts/:id/installments`
- `POST /api/financial-debts/:id/installments/bulk`
- `PUT /api/financial-installments/:id`
- `DELETE /api/financial-installments/:id`

Finans borç ödemeleri:

- `POST /api/financial-debt-payments`
- `GET /api/financial-debt-payments`
- `GET /api/financial-debt-payments?debt_id=1`
- `GET /api/financial-debt-payments?installment_id=1`
- `DELETE /api/financial-debt-payments/:id`

Finans uyarıları:

- `GET /api/financial-alerts`

Tarihe göre borç raporu:

- `GET /api/debt-snapshot?date=2026-06-02`

Cüzdan:

- `GET /api/wallet/summary`
- `GET /api/wallet/transactions`
- `POST /api/wallet/transactions`
- `DELETE /api/wallet/transactions/:id`

Banka cüzdanı:

- `GET /api/bank-accounts`
- `POST /api/bank-accounts`
- `GET /api/bank-accounts/:id`
- `PUT /api/bank-accounts/:id`
- `DELETE /api/bank-accounts/:id`
- `GET /api/bank-accounts/:id/transactions`
- `POST /api/bank-accounts/:id/transactions`
- `DELETE /api/bank-transactions/:id`
- `GET /api/bank-wallet/summary`
- `GET /api/bank-wallet/daily-summary?date=2026-06-09`
- `GET /api/bank-wallet/monthly-summary?month=2026-06`

Ürün/stok modülü:

- `POST /api/products`
- `GET /api/products`
- `GET /api/products/:id`
- `PUT /api/products/:id`
- `DELETE /api/products/:id`
- `POST /api/stock-movements`
- `GET /api/stock-movements`
- `GET /api/products/:id/stock`

## Örnek POST Bodyleri

Firma:

```json
{
  "name": "ABC Gıda",
  "phone": "05550000000",
  "address": "İstanbul",
  "note": "Haftalık tedarikçi",
  "is_active": true
}
```

Firma gelen fatura:

```json
{
  "supplier_id": 1,
  "transaction_date": "2026-06-03",
  "type": "invoice",
  "amount": "25000",
  "invoice_no": "FTR-123",
  "note": "Ürün faturası"
}
```

Dövizli firma gelen faturası:

```json
{
  "supplier_id": 1,
  "transaction_date": "2026-06-20",
  "type": "invoice",
  "amount": "1000",
  "currency": "USD",
  "exchange_rate": "32.50",
  "invoice_no": "USD-2026-01",
  "note": "Dolar bazlı ürün faturası"
}
```

`exchange_rate` USD/EUR için isteğe bağlıdır. Gönderilmezse Frankfurter'dan güncel TRY kuru alınır; sağlayıcı çalışmazsa veritabanındaki son başarılı kur kullanılır. Kur bulunamazsa istemci manuel kur göndermelidir.

Kur sorgusu:

```bash
curl -H "Authorization: Bearer TOKEN" \
  "http://localhost:8081/api/exchange-rates/latest?currency=USD"
```

Firmaya ödeme:

```json
{
  "supplier_id": 1,
  "transaction_date": "2026-06-03",
  "type": "payment",
  "amount": "10000",
  "payment_method": "cash",
  "note": "Nakit ödeme"
}
```

Firma iade / fatura düşümü:

```json
{
  "supplier_id": 1,
  "transaction_date": "2026-06-03",
  "type": "return",
  "amount": "1500",
  "note": "İade ürün düşümü"
}
```

Firma hareketi dosya yükleme:

```bash
curl -X POST http://localhost:8081/api/supplier-transactions/1/files \
  -H "Authorization: Bearer TOKEN" \
  -F "files=@/path/to/fatura-1.jpg" \
  -F "files=@/path/to/fatura-2.pdf"
```

`invoice`, `payment` ve `return` hareketlerine JPG, JPEG, PNG, WEBP veya PDF dosyası eklenebilir. Aynı dosya alanı, ödeme makbuzu/dekont ve iade faturası evrakı için de kullanılır. `payment` ve `return` kayıtlarında ilk ilişkili dosya ayrıca hareket cevabındaki `image_url` ve `file_path` alanlarına yazılır.

Firma hareketi oluştururken dosya ekleme:

```bash
curl -X POST http://localhost:8081/api/supplier-transactions \
  -H "Authorization: Bearer TOKEN" \
  -F "supplier_id=1" \
  -F "transaction_date=2026-06-03" \
  -F "type=payment" \
  -F "amount=10000" \
  -F "payment_method=cash" \
  -F "note=Nakit ödeme" \
  -F "files=@/path/to/dekont.pdf"
```

Firma hareketi düzenlerken dosya değiştirme veya silme:

```bash
curl -X PUT http://localhost:8081/api/supplier-transactions/1 \
  -H "Authorization: Bearer TOKEN" \
  -F "supplier_id=1" \
  -F "transaction_date=2026-06-03" \
  -F "type=return" \
  -F "amount=1500" \
  -F "note=İade ürün düşümü" \
  -F "replace_files=true" \
  -F "files=@/path/to/iade-faturasi.jpg"
```

Belirli dosyaları silmek için düzenleme isteğinde `delete_file_ids=1,2` gönderilebilir; tekil silme endpoint'i de geriye uyumlu olarak çalışır.

Firma hareketi dosyalarını listeleme:

```bash
curl http://localhost:8081/api/supplier-transactions/1/files \
  -H "Authorization: Bearer TOKEN"
```

Firma hareketi dosyası silme:

```bash
curl -X DELETE http://localhost:8081/api/supplier-transaction-files/1 \
  -H "Authorization: Bearer TOKEN"
```

Fatura dosyasını görüntüleme:

```bash
curl http://localhost:8081/uploads/invoices/DOSYA_ADI.pdf \
  -H "Authorization: Bearer TOKEN"
```

Personel:

```json
{
  "name": "Mehmet Yılmaz",
  "phone": "05551112233",
  "daily_wage": "850",
  "is_active": true,
  "note": "Kasiyer"
}
```

Personel çalışma günü:

```json
{
  "employee_id": 1,
  "transaction_date": "2026-05-30",
  "type": "work",
  "work_days": "1",
  "note": "Tam gün"
}
```

Personel ödeme veya avans:

```json
{
  "employee_id": 1,
  "transaction_date": "2026-05-30",
  "type": "advance",
  "amount": "1000",
  "note": "Avans"
}
```

Günlük kasa:

```json
{
  "report_date": "2026-05-30",
  "cash_amount": "15000",
  "pos_amount": "22500.50",
  "qr_amount": "3000",
  "credit_collection": "1200",
  "credit_given": "750",
  "note": "Cumartesi kasa"
}
```

Gider:

```json
{
  "expense_date": "2026-05-30",
  "category": "elektrik",
  "amount": "2400",
  "payment_method": "bank",
  "note": "Fatura"
}
```

Gelir:

```json
{
  "income_date": "2026-05-30",
  "category": "veresiye_tahsilat",
  "amount": "1800",
  "payment_method": "cash",
  "note": "Tahsilat"
}
```

Finans borcu:

```json
{
  "debt_type": "bank_loan",
  "institution_name": "Ziraat Bankası",
  "title": "İşletme kredisi",
  "total_amount": "250000",
  "start_date": "2026-05-01",
  "end_date": "2027-04-30",
  "status": "active",
  "note": "Aylık kredi takibi"
}
```

Finans taksiti:

```json
{
  "installment_no": 1,
  "due_date": "2026-06-30",
  "amount": "22500",
  "note": "1. taksit"
}
```

Toplu finans taksiti:

```json
{
  "installments": [
    {
      "installment_no": 1,
      "due_date": "2026-06-30",
      "amount": "22500",
      "note": "1. taksit"
    },
    {
      "installment_no": 2,
      "due_date": "2026-07-30",
      "amount": "21800",
      "note": "2. taksit"
    }
  ]
}
```

Finans borcu ödemesi:

```json
{
  "financial_debt_id": 1,
  "installment_id": 1,
  "payment_date": "2026-06-01",
  "amount": "22500",
  "payment_method": "bank_transfer",
  "note": "1. taksit"
}
```

Tarihe göre borç raporu:

```bash
curl -H "Authorization: Bearer TOKEN" \
  "https://localhost:8081/api/debt-snapshot?date=2026-06-02"
```

Para analizi / Para nerede raporu:

```bash
curl -H "Authorization: Bearer TOKEN" \
  "https://localhost:8081/api/money-analysis?month=2026-06"
```

## Finans Merkezi

Finans Merkezi endpointleri JWT korumalıdır. Eski rapor endpointleri geriye uyumluluk için çalışmaya devam eder; yeni ekranlar tüm finans analizlerini aşağıdaki gruptan kullanmalıdır.

```bash
curl -H "Authorization: Bearer TOKEN" http://localhost:8081/api/finance-center/summary
curl -H "Authorization: Bearer TOKEN" "http://localhost:8081/api/finance-center/history?date=2026-06-02"
curl -H "Authorization: Bearer TOKEN" "http://localhost:8081/api/finance-center/money-flow?start_date=2026-06-01&end_date=2026-06-30"
curl -H "Authorization: Bearer TOKEN" http://localhost:8081/api/finance-center/debt-distribution
curl -H "Authorization: Bearer TOKEN" http://localhost:8081/api/finance-center/cashflow
curl -H "Authorization: Bearer TOKEN" http://localhost:8081/api/finance-center/alerts
```

- `summary`: güncel kasa, banka, borç ve net durumu döner.
- `summary` ayrıca `cash_balance`, `bank_balance`, `total_money`, `supplier_debts`, `financial_debts`, `personnel_debts`, `total_debts`, `net_worth`, `monthly_revenue`, `monthly_collected`, `monthly_expense` ve `monthly_net_cashflow` alanlarını döner. `monthly_details` satış, tahsilat ve gider kalemlerini ayrı gösterir.
- `history`: seçilen gün sonundaki kasa/banka ve borç durumunu hareketlerden hesaplar.
- `money-flow`: tarih aralığındaki gelir, gider, ödeme kırılımları, beklenen ve gerçek bakiye farkını döner. Tarihler verilmezse mevcut ayın başından bugüne kadar kullanılır.
- `cashflow`: son 12 ayın gelir, gider ve net kazanç serisini döner.

Banka hesabı:

```json
{
  "account_name": "Ana Banka Hesabı",
  "bank_name": "Garanti",
  "iban": "",
  "opening_balance": "20000"
}
```

Banka hareketi:

```json
{
  "transaction_date": "2026-06-09",
  "transaction_type": "pos_income",
  "amount": "10000",
  "title": "POS Yatışı",
  "description": "08.06 POS ertesi gün hesaba geçti"
}
```

Banka cüzdanı özet cevabı:

```json
{
  "total_balance": "30000",
  "accounts": [
    {
      "id": 1,
      "account_name": "Ana Banka",
      "bank_name": "Garanti",
      "current_balance": "30000"
    }
  ],
  "today_income": "15000",
  "today_outcome": "10000",
  "today_net": "5000"
}
```

Cüzdan hareketi:

```json
{
  "transaction_date": "2026-06-10",
  "transaction_type": "cash_income",
  "amount": "10000",
  "title": "Nakit Gelir",
  "description": "Gün sonu kasa girişi"
}
```

Cüzdan açılış bakiyesi:

```json
{
  "transaction_date": "2026-06-10",
  "transaction_type": "opening_balance",
  "amount": "20000",
  "title": "Açılış Bakiyesi",
  "description": "Cüzdan başlangıç bakiyesi"
}
```

## Hesap Mantığı

- Firma borcu: `invoice ve eski purchase toplamı - payment toplamı - return toplamı`. TL hesapları her zaman hareketin kayıt anındaki `amount_try` değeriyle yapılır.
- Firma hareket para birimleri: `TRY`, `USD`, `EUR`. Eski hareketler migration ile `TRY`, kur `1`, `amount_original = amount` ve `amount_try = amount` olarak korunur.
- TRY harekette `exchange_rate = 1`; USD/EUR harekette `amount_try = amount_original x exchange_rate` backend tarafından hesaplanır. İstemciden gelen `amount_try` dikkate alınmaz.
- Firma bakiye endpointleri `try`, `usd`, `eur` ve `total_try` döviz toplamlarını da döner.
- Firma hareket tipleri: `invoice` gelen fatura, `payment` ödeme, `return` iade/fatura düşümü. Eski `purchase` kayıtları `invoice` gibi hesaplanır.
- Firma ödeme yöntemleri: `cash`, `credit_card`, `current_account`, `bank_transfer`, `other`
- Personel borcu: `(work_days toplamı x daily_wage) - payment toplamı - advance toplamı`
- Günlük ciro: `cash_amount + pos_amount + qr_amount`
- Dashboard gelirleri: günlük kasa cirosu + gelir kayıtları
- Finans borcu kalan tutar: `financial_debt_installments.amount toplamı - financial_debt_payments.amount toplamı`
- Taksit ödemesinde ilgili taksitin `paid_amount` değeri tekrar hesaplanır
- `paid_amount = amount` ise taksit `paid` olur
- `paid_amount > 0` ve `amount` değerinden küçükse taksit `partial_paid` olur
- `due_date < today` ve tam ödenmemişse taksit `overdue` olur
- Yaklaşan finans taksitleri: önümüzdeki 30 gün içinde vadesi gelen ödenmemiş taksitler
- Banka hareketlerinde `cash_deposit`, `pos_income`, `bank_income`, `transfer_in` bakiyeyi artırır; `payment`, `expense`, `transfer_out` bakiyeyi azaltır.
- Banka `correction` hareketinde tutar pozitifse bakiye artar, negatifse azalır.
- Banka hareketi silinirse hesabın `current_balance` ve hareketlerin `balance_after` değerleri tüm hareketlerden tekrar hesaplanır.
- Cüzdan bakiyesi tüm geçmiş `wallet_transactions` hareketlerinden hesaplanır ve gün değişince sıfırlanmaz.
- Cüzdan girişleri: `opening_balance`, `cash_income`, `cash_sale`, `pos_income`, `bank_income`, `cash_deposit`
- Cüzdan çıkışları: `payment`, `expense`, `cash_withdraw`
- Cüzdan `correction` hareketinde tutar pozitifse bakiye artar, negatifse azalır.
- Para analizi gelirini günlük kasa cirosu ve `income_entries` kayıtlarından hesaplar. Gider tarafına gider kayıtları, yalnızca `cash` yöntemli firma ödemeleri ve personel/finans ödemeleri eklenir. Banka, kart ve havale ile yapılan firma ödemeleri nakit çıkışı sayılmaz. Cüzdan ve banka hareketleri para transferi olabildiği için gelir-gidere tekrar eklenmez; güncel para konumu olarak `cash_balance` ve `bank_balance` alanlarında gösterilir.
- Para analizi `employee_advances` alanı seçilen ay içindeki personel avanslarını gösterir. Müşteri cari tablosu yoksa `customer_receivables`, POS bekleyen tahsilat için ayrı kayıt yoksa `pending_pos` sıfır döner.
- Net: `gelir - gider`
