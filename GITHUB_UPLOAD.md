# Atualizar um repositório FirebirdToolbox já existente para v1.2.10

Se o repositório já está publicado, copie os arquivos desta versão sobre a árvore local e execute:

```powershell
git add .
git status
git commit -m "fix(ci): corrigir ajuda e atualizar GitHub Actions v1.2.10"
git push origin main
```

Depois que o CI ficar verde, publique a tag:

```powershell
git tag -a v1.2.10 -m "Firebird Toolbox v1.2.10"
git push origin v1.2.10
```

---

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
git commit -m "chore(repo): preparar Firebird Toolbox v1.2.10 para GitHub"
```

## 4. Conecte ao GitHub

Substitua `SEU_USUARIO`:

```powershell
git remote add origin https://github.com/SEU_USUARIO/FirebirdToolbox.git
git push -u origin main
```

## 5. Publicar a v1.2.10

```powershell
git tag -a v1.2.10 -m "Firebird Toolbox v1.2.10"
git push origin v1.2.10
```

O workflow `.github/workflows/release.yml` será iniciado automaticamente e criará a Release com executáveis x64/x86 e checksums SHA-256.

> Como a versão 1.2.10 já existe antes deste repositório, a primeira execução do workflow recompilará os binários a partir do código versionado. Para versões futuras, atualize `appVersion`/changelog normalmente e publique a nova tag.

## 6. Configurar assinatura digital (recomendado para distribuição pública)

Se você tiver um certificado de assinatura de código em `.pfx`, converta-o para Base64 localmente e grave somente o valor nos **GitHub Secrets** do repositório. Nunca faça commit do `.pfx`.

No PowerShell:

```powershell
[Convert]::ToBase64String([IO.File]::ReadAllBytes("C:\caminho\certificado.pfx")) | Set-Clipboard
```

No GitHub, crie:

- `WINDOWS_SIGN_CERT_BASE64`;
- `WINDOWS_SIGN_CERT_PASSWORD`.

A próxima tag será compilada com VersionInfo/manifesto e, se os dois secrets estiverem presentes, os EXEs serão assinados e verificados antes da geração do ZIP e dos hashes. A Release também publica `SIGNATURE_STATUS.txt`.
