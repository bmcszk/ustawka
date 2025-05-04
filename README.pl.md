# Ustawka

Aplikacja webowa do śledzenia polskich aktów prawnych z API Sejmu.

## Funkcje

- Przeglądanie aktów prawnych w formie tablicy Kanban
- Filtrowanie aktów według roku (2021-obecnie)
- Kategoryzacja aktów według statusu:
  - W przygotowaniu
  - Uchylone
  - Obowiązujące
- Przeglądanie szczegółowych informacji o każdym akcie

## Technologie

- Backend: Go
- Frontend: HTML, TailwindCSS, HTMX
- API: API Sejmu (https://api.sejm.gov.pl)

## Wymagania

- Go 1.24.2 lub nowszy
- Nowoczesna przeglądarka internetowa
- Make (opcjonalnie, do używania Makefile)

## Instalacja

1. Sklonuj repozytorium:
```bash
git clone https://github.com/bmcszk/ustawka.git
cd ustawka
```

2. Zainstaluj zależności:
```bash
make deps
```

3. Uruchom aplikację:
```bash
make run
```

Aplikacja będzie dostępna pod adresem http://localhost:8080

## Rozwój

### Używanie Makefile

Projekt zawiera plik Makefile z typowymi zadaniami deweloperskimi:

```bash
make build      # Budowanie aplikacji
make run        # Uruchomienie aplikacji
make test       # Uruchomienie wszystkich testów
make test-unit  # Uruchomienie tylko testów jednostkowych
make test-e2e   # Uruchomienie tylko testów end-to-end
make clean      # Czyszczenie plików budowania
make deps       # Instalacja zależności
make help       # Wyświetlenie wszystkich dostępnych komend
```

### Testowanie

Projekt zawiera dwa typy testów:
- Testy jednostkowe: Testują poszczególne komponenty w izolacji
- Testy end-to-end: Testują aplikację z rzeczywistym API Sejmu

Aby uruchomić konkretny typ testów:
```bash
make test-unit  # Uruchom tylko testy jednostkowe
make test-e2e   # Uruchom tylko testy end-to-end
make test       # Uruchom wszystkie testy
```

### Struktura Projektu

```
