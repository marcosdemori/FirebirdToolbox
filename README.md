# 🔥 Firebird Toolbox

Utilitário portátil para Windows destinado à **manutenção, correção, recuperação e restauração de bancos Firebird 1.0 a 5.0**.

**Versão atual:** 1.2.10  

A v1.2.10 mantém o hardening de distribuição Windows da v1.2.9 e corrige o CI/Release: a Ajuda integrada documenta explicitamente `gbak -inactive` no saneamento de chaves duplicadas, e os workflows usam Actions atuais baseados em Node 24.

**Projeto feito por Marcos De Mori Laiola**

## Recursos principais

- interface gráfica Win32 nativa e portátil;
- suporte a Firebird 1.0, 1.5, 2.0, 2.1, 2.5, 3.0, 4.0 e 5.0;
- `Corrigir banco`;
- `Corrigir banco (avançado)` para cenários de corrupção;
- `Somente restaurar`;
- seleção independente da versão de origem e da versão de restauração;
- detecção de instalação, serviço e porta Firebird;
- log integrado resumido ou detalhado;
- diagnóstico com `gfix`, backup/restore com `gbak` e conferência com `gstat`;
- cópia de trabalho e relatório de recuperação em rotinas avançadas;
- tratamento de corrupção de índices, páginas e conflitos de chave;
- tentativa de resgate de registros antes de excluir uma relação inteira;
- restore de teste antes de considerar um backup recuperado como válido;
- executáveis Windows x64 e x86.

> **Importante:** recuperação de banco corrompido nunca deve ser executada sobre o único arquivo disponível. Preserve cópias externas antes de iniciar qualquer procedimento.

## Download

Os executáveis compilados devem ser disponibilizados na seção **Releases** deste repositório.

Ao criar uma tag no formato `v1.2.10`, por exemplo, o workflow de release compila automaticamente:

- Windows x64;
- Windows x86 / 32-bit;
- arquivo SHA-256;
- status explícito da assinatura Authenticode.

Os binários são anexados diretamente à GitHub Release e não ficam armazenados no histórico Git.

## Rotinas

### Corrigir banco

Fluxo voltado à manutenção normal: validação, sweep, backup e, quando selecionado, restauração para a versão escolhida.

### Corrigir banco (avançado)

Fluxo protegido para corrupção severa. Trabalha com cópia de recuperação, diagnóstico profundo, tentativa de preservação dos dados, restore de teste e relatório técnico.

A rotina avançada evita executar reparos destrutivos diretamente no arquivo original quando o fluxo de proteção está disponível.

### Somente restaurar

Restaura um backup Firebird para o destino selecionado sem executar a etapa de correção do banco de origem.

## Compilar localmente

### Requisitos

- Windows;
- Go 1.23 ou superior compatível;
- acesso à internet na primeira execução para instalar `go-winres v0.3.3`;
- Git, opcional para desenvolvimento.

Execute:

```powershell
.\tools\build.ps1
```

Os arquivos serão gerados em `dist/`.

Para compilar somente x64:

```powershell
$env:GOOS = "windows"
$env:GOARCH = "amd64"
.\tools\prepare-resources.ps1 -Version "1.2.10"
$env:GOOS = "windows"
$env:GOARCH = "amd64"
go build -trimpath -buildvcs=false -ldflags="-H=windowsgui -X main.appVersion=1.2.10" -o FirebirdToolbox.exe .
```

## Publicar no GitHub

Consulte [GITHUB_UPLOAD.md](GITHUB_UPLOAD.md) ou execute `PUBLICAR_GITHUB.bat` para um fluxo guiado com confirmação antes do commit e do push.

## GitHub Actions

O projeto possui dois workflows:

- **CI:** valida `gofmt`, executa `go vet`, testes e builds x64/x86 em push/PR para `main`;
- **Release:** em tags `v*.*.*`, gera recursos nativos do Windows, compila x64/x86, assina com Authenticode quando o certificado estiver configurado, calcula SHA-256 após a assinatura e cria a GitHub Release automaticamente.

A CI não envia artifacts intermediários, evitando consumo desnecessário de Actions Storage.

## Estrutura

```text
.
├── .github/workflows/      # CI e Release automática
├── assets/                 # ícones incorporados
├── docs/                   # documentação complementar
├── scripts/                # BATs originais de referência
├── winres/                 # ícone, manifesto e VersionInfo do executável
├── tools/build.ps1         # build local x64/x86
├── tools/sign-windows.ps1  # assinatura Authenticode opcional
├── PUBLICAR_GITHUB.bat     # publicacao guiada opcional
├── main.go                 # aplicação Win32
├── main_test.go            # regressões
├── go.mod
├── LEIA-ME.md              # ajuda incorporada ao EXE
├── README_FirebirdToolbox.md
├── CHANGELOG.md
├── SECURITY.md
└── LICENSE
```

## Documentação detalhada

Consulte [README_FirebirdToolbox.md](README_FirebirdToolbox.md) para o histórico e os detalhes técnicos das rotinas de recuperação.

O conteúdo exibido pelo botão **Ajuda** do aplicativo é mantido em [LEIA-ME.md](LEIA-ME.md).

## Privacidade e segurança

Não publique bancos reais, backups, senhas SYSDBA ou relatórios contendo dados de clientes. Consulte [SECURITY.md](SECURITY.md) e [docs/DISTRIBUICAO_WINDOWS.md](docs/DISTRIBUICAO_WINDOWS.md).

## Licença

Copyright © 2026 Marcos De Mori Laiola. Todos os direitos reservados. Consulte [LICENSE](LICENSE).
