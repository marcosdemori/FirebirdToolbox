# 🔥 Firebird Toolbox

Utilitário portátil para Windows destinado à **manutenção, correção, recuperação e restauração de bancos Firebird 1.0 a 5.0**.

**Versão atual:** 1.2.20

A v1.2.20 passa a distribuir **um único executável Windows x86/x64**. O binário é PE32/x86 e funciona em Windows 32-bit e em Windows 64-bit via WOW64. A descoberta do Firebird deixa de depender de caminhos, portas ou nomes de serviço usados na máquina de desenvolvimento: o Toolbox consulta o Registry oficial do Firebird, Program Files/ProgramW6432, serviços Windows e PATH, confirma a versão pelo `gbak -z` e lê `RemoteServicePort` do `firebird.conf` da instalação encontrada. Os BATs incorporados são apenas referência histórica e não fornecem o nome do serviço usado pela execução.

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
- tratamento de corrupção de índices, páginas e conflitos de chave, inclusive múltiplas duplicidades em sequência;
- tentativa de resgate de registros antes de excluir uma relação inteira;
- fallback de reconstrução limpa de relação no Firebird 3.0+ quando uma PK/UNIQUE continua inativa sem nova duplicidade detectável;
- restore de teste antes de considerar um backup recuperado como válido;
- pós-recuperação com `RELATORIO_SAUDE_PERFORMANCE.txt` e gate explícito `APTO PARA PRODUÇÃO`;
- um único executável Windows x86/x64;

> **Importante:** recuperação de banco corrompido nunca deve ser executada sobre o único arquivo disponível. Preserve cópias externas antes de iniciar qualquer procedimento.

## Download

Os executáveis compilados devem ser disponibilizados na seção **Releases** deste repositório.

Ao criar uma tag no formato `v1.2.20`, o workflow de release compila automaticamente um único `FirebirdToolbox_v1.2.20_x86_x64.exe`, compatível com Windows x86 e x64.
- arquivo SHA-256;
- assinatura Authenticode obrigatória via Microsoft Artifact Signing;
- timestamp RFC3161;
- atestação de proveniência do GitHub;
- arquivo SHA-256 calculado após a assinatura.

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

Para teste local sem assinatura, é necessário assumir explicitamente que o binário é apenas de desenvolvimento:

```powershell
.\tools\build.ps1 -Version "1.2.20" -AllowUnsignedDev
```

O arquivo recebe `_UNSIGNED_DEV` no nome. Para distribuição pública use a Release oficial assinada.

Para compilar manualmente o executável único x86/x64:

```powershell
$env:GOOS = "windows"
$env:GOARCH = "386"
.\tools\prepare-resources.ps1 -Version "1.2.20"
go build -trimpath -buildvcs=false -ldflags="-H=windowsgui -X main.appVersion=1.2.20" -o FirebirdToolbox_v1.2.20_x86_x64.exe .
```

## Publicar no GitHub

Consulte [GITHUB_UPLOAD.md](GITHUB_UPLOAD.md) ou execute `PUBLICAR_GITHUB.bat` para um fluxo guiado com confirmação antes do commit e do push.

## GitHub Actions

O projeto possui dois workflows:

- **CI:** valida `gofmt`, testa compatibilidade x64/x86 e verifica o executável único x86/x64 em push/PR para `main`;
- **Release:** em tags `v*.*.*`, exige configuração do Microsoft Artifact Signing, compila o executável único x86/x64, assina e aplica timestamp, valida Authenticode, executa preflight do Defender quando disponível, calcula SHA-256, gera atestação e só então cria a GitHub Release.

A CI não envia artifacts intermediários, evitando consumo desnecessário de Actions Storage.

## Estrutura

```text
.
├── .github/workflows/      # CI e Release automática
├── assets/                 # ícones incorporados
├── docs/                   # documentação complementar
├── scripts/                # BATs originais de referência
├── winres/                 # ícone, manifesto e VersionInfo do executável
├── tools/build.ps1         # build local único x86/x64
├── tools/sign-windows.ps1  # assinatura PFX apenas para build local
├── tools/defender-preflight.ps1 # preflight Defender da Release
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
