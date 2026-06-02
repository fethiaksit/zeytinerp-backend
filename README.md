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
```

Production ortamında `JWT_SECRET` boş bırakılamaz. Development ortamında boşsa geçici development secret kullanılır, fakat gerçek kullanımda `.env` içine güçlü bir değer yazılmalıdır.

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

Giriş örneği:

```bash
curl -X POST http://localhost:8081/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"123456"}'
```

Token ile istek örneği:

```bash
curl http://localhost:8081/api/dashboard \
  -H "Authorization: Bearer JWT_TOKEN"
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
- `monthly_financial_due`
- `overdue_financial_count`
- `upcoming_financial_due_7_days`
- `upcoming_financial_due_30_days`
- `nearest_financial_installments`

Firmalar:

- `POST /api/suppliers`
- `GET /api/suppliers`
- `GET /api/suppliers/:id`
- `PUT /api/suppliers/:id`
- `DELETE /api/suppliers/:id`
- `GET /api/suppliers/:id/balance`
- `GET /api/suppliers-balances`

Firma hareketleri:

- `POST /api/supplier-transactions`
- `GET /api/supplier-transactions`
- `GET /api/supplier-transactions?supplier_id=1`
- `DELETE /api/supplier-transactions/:id`

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
- `PUT /api/expenses/:id`
- `DELETE /api/expenses/:id`

Gelirler:

- `POST /api/income-entries`
- `GET /api/income-entries`
- `GET /api/income-entries?start_date=2026-05-01&end_date=2026-05-31`
- `PUT /api/income-entries/:id`
- `DELETE /api/income-entries/:id`

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

Firma borç girişi:

```json
{
  "supplier_id": 1,
  "transaction_date": "2026-05-30",
  "type": "purchase",
  "amount": "12500.75",
  "payment_method": "cash",
  "note": "Haftalık mal alımı"
}
```

Firmaya ödeme:

```json
{
  "supplier_id": 1,
  "transaction_date": "2026-05-30",
  "type": "payment",
  "amount": "5000",
  "payment_method": "bank",
  "note": "Kısmi ödeme"
}
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

## Hesap Mantığı

- Firma borcu: `purchase toplamı - payment toplamı`
- Personel borcu: `(work_days toplamı x daily_wage) - payment toplamı - advance toplamı`
- Günlük ciro: `cash_amount + pos_amount + qr_amount`
- Dashboard gelirleri: günlük kasa cirosu + gelir kayıtları
- Finans borcu kalan tutar: `financial_debt_installments.amount toplamı - financial_debt_payments.amount toplamı`
- Taksit ödemesinde ilgili taksitin `paid_amount` değeri tekrar hesaplanır
- `paid_amount = amount` ise taksit `paid` olur
- `paid_amount > 0` ve `amount` değerinden küçükse taksit `partial_paid` olur
- `due_date < today` ve tam ödenmemişse taksit `overdue` olur
- Yaklaşan finans taksitleri: önümüzdeki 30 gün içinde vadesi gelen ödenmemiş taksitler
- Net: `gelir - gider`
