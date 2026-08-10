# Publicar este projeto no GitHub

## 1. Crie um repositório vazio

No GitHub, crie `FirebirdToolbox` sem adicionar README, .gitignore ou licença pela interface.

## 2. Abra PowerShell nesta pasta

```powershell
git init
git branch -M main
git add .
git status
```

Confira especialmente se **não** aparecem bancos, backups, `.env`, arquivos `RECOVERY_...` ou dados de clientes.

## 3. Primeiro commit

```powershell
git commit -m "chore(repo): preparar Firebird Toolbox v1.2.8 para GitHub"
```

## 4. Conecte ao GitHub

Substitua `SEU_USUARIO`:

```powershell
git remote add origin https://github.com/SEU_USUARIO/FirebirdToolbox.git
git push -u origin main
```

## 5. Publicar a v1.2.8

```powershell
git tag -a v1.2.8 -m "Firebird Toolbox v1.2.8"
git push origin v1.2.8
```

O workflow `.github/workflows/release.yml` será iniciado automaticamente e criará a Release com executáveis x64/x86 e checksums SHA-256.

> Como a versão 1.2.8 já existe antes deste repositório, a primeira execução do workflow recompilará os binários a partir do código versionado. Para versões futuras, atualize `appVersion`/changelog normalmente e publique a nova tag.
