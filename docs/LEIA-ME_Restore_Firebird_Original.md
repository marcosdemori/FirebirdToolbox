# LEIA-ME — Restauração de Backup no Firebird **2.5** e **5.0** (Windows)

Este documento explica como restaurar backups (`.fbk`/`.gbk`/`.bak`) do Firebird, separando claramente os passos para **Firebird 2.5** e **Firebird 5.0**.  
Inclui também dicas, verificação de dialeto e solução de problemas comuns.

> **Importante:** Verifique se o serviço do Firebird está iniciado.

---

## 📂 Scripts incluídos (exemplos)

- `Somente Restore 5_0.bat` — Restauração no **Firebird 5.0** via Service Manager
- `Somente Restore 3_0.bat` — Restauração para versões 3.0
- `Corrigir Banco_3_0.bat` — Ajustes/rotinas específicas para **3.0**
- `Corrigir Banco_2_5.bat` — Ajustes/rotinas específicas para **2.5**

> Os nomes dos arquivos podem variar; ajuste as instruções conforme os scripts que você tem.

---

## 🔧 Parâmetros que você **SEMPR**E ajusta nos scripts

Abra o `.bat` no Bloco de Notas e confira/ajuste no início do arquivo:

```bat
SET FIREBIRD_PATH=C:\Program Files\Firebird\Firebird_5_0
SET BACKUP_FILE=C:\Caminho\para\seu\backup\MeuBackup.BAK
SET DEST_DB=C:\Caminho\para\salvar\Banco.RED  :: ou .FDB, conforme seu padrão
SET ISC_USER=SYSDBA
SET ISC_PASSWORD=masterkey
```

**Dicas:**
- Se o caminho tiver espaços, mantenha as aspas (o script já cuida disso).
- `DEST_DB` **não pode** apontar para um arquivo aberto por outro processo (ex.: aplicativo conectado).
- Faça **backup do banco atual** antes de sobrescrever qualquer arquivo existente.

---

## 🧩 Firebird **2.5** — Como restaurar

### Pré‑requisitos
- Firebird 2.5 instalado e serviço ativo.
- Backup gerado por `gbak` compatível.

### Passo a passo (padrão local)
1. **Pare** qualquer aplicação que esteja usando o banco de destino.
2. Abra o `.bat` (ex.: `Corrigir Banco_2_5.bat` se você tiver) e ajuste os parâmetros.
3. Aguarde a mensagem de **sucesso**.

### Comando de referência (linha única)
```bat
"C:\Program Files\Firebird\Firebird_2_5\bin\gbak.exe" -c -v ^
  -user SYSDBA -password masterkey ^
  "C:\Backups\DBCOM_YYYYMMDD_HHMMSS.BAK" "C:\Bancos\DBCOM25.FDB"
```

### Ajustando/checando **dialeto 1** no 2.5 (opcional)
```bat
"C:\Program Files\Firebird\Firebird_2_5\bin\gfix.exe" -user SYSDBA -password masterkey -sql_dialect 1 "C:\Bancos\DBCOM25.FDB"
"C:\Program Files\Firebird\Firebird_2_5\bin\gstat.exe" -h "C:\Bancos\DBCOM25.FDB"
```

> Procure “**dialect**” no header para conferir.

---

## 🧩 Firebird **5.0** — Como restaurar

### Por que é diferente?
No 5.0, recomenda‑se usar o **Service Manager** (via `localhost:service_mgr`). Além disso, o **`gbak` 5.0 não aceita `-sql_dialect`**; o dialeto é ajustado posteriormente com `gfix`.

### Passo a passo
1. **Pare** qualquer aplicação que esteja usando o arquivo destino.
2. Edite `Somente Restore 5_0.bat` (ou seu script equivalente) e ajuste os parâmetros:
   ```bat
   SET FIREBIRD_PATH=C:\Program Files\Firebird\Firebird_5_0
   SET BACKUP_FILE=...
   SET DEST_DB=...
   SET ISC_USER=SYSDBA
   SET ISC_PASSWORD=masterkey
   ```
3. Execute.