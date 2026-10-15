# 🔥 Firebird Toolbox

Utilitário portátil para Windows destinado à **manutenção, correção, recuperação e restauração de bancos Firebird 1.0 a 5.0**.

**Versão atual:** 1.2.32

A v1.2.32 trata o restore que falha ao recriar uma `FOREIGN KEY` porque os registros perdidos na corrupção eram linhas-pai e as linhas-filhas sobreviveram apontando para chaves inexistentes. O utilitário detecta o erro `Non-existent PRIMARY or UNIQUE KEY specified for FOREIGN KEY`, extrai o DDL das constraints para `CONSTRAINTS_ORIGINAIS.sql`, tenta o restore com `-n` (`-no_validity`) e grava em `LINHAS_ORFAS.sql` as consultas que listam os filhos sem pai, para inspeção antes de qualquer exclusão. O banco restaurado dessa forma não tem `FOREIGN KEY` nem constraints `CHECK` e nunca recebe `APTO PARA PRODUÇÃO`.

A versão mantém **um único executável Windows x86/x64**, preserva o ícone oficial no arquivo/janela/barra de tarefas e o acabamento da interface: os atalhos `Abrir relatório` e `Localizar banco` ganham destaque quando ficam disponíveis, e a borda superior dos blocos fica contínua acima do ícone e do título, sem ser apagada pela faixa branca do texto. A área de execução mantém as ações principais na primeira linha e os atalhos de resultado em uma segunda linha, sem truncar os textos nem aumentar a janela. A interface também deixa os atalhos de resultado explícitos: `Abrir pasta` abre a pasta geral, `Abrir relatório` abre o relatório gerado e `Localizar banco` abre o Explorer selecionando somente o banco final produzido na execução. O rodapé usa a assinatura discreta `Feito por Marcos De Mori Laiola`. As rotinas continuam: `Manutenção / conversão` para bancos saudáveis e migração de versão, `Recuperar banco corrompido` para corrupção real e `Somente restaurar` para restore direto. Mantém a pré-validação antes da execução, progresso por fases, cancelamento seguro, resumo final objetivo e atalhos para Localizar banco/Abrir relatório/Abrir pasta. Caminhos e escolhas não sensíveis podem ser restaurados na próxima abertura; a senha nunca é persistida. O acabamento branco sem rastros da v1.2.24 permanece. O binário é PE32/x86 e funciona em Windows 32-bit e em Windows 64-bit via WOW64. A descoberta do Firebird deixa de depender de caminhos, portas ou nomes de serviço usados na máquina de desenvolvimento: o Toolbox consulta o Registry oficial do Firebird, Program Files/ProgramW6432, serviços Windows e PATH, confirma a versão pelo `gbak -z` e lê `RemoteServicePort` do `firebird.conf` da instalação encontrada. Os BATs incorporados são apenas referência histórica e não fornecem o nome do serviço usado pela execução.

**Feito por Marcos De Mori Laiola**

## Recursos principais

- interface gráfica Win32 nativa e portátil;
- suporte a Firebird 1.0, 1.5, 2.0, 2.1, 2.5, 3.0, 4.0 e 5.0;
- `Manutenção / conversão`;
- `Recuperar banco corrompido` para cenários de corrupção;
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

Ao criar uma tag no formato `v1.2.32`, o workflow de release compila automaticamente um único `FirebirdToolbox_v1.2.32_x86_x64.exe`, compatível com Windows x86 e x64.
- arquivo SHA-256;
- assinatura Authenticode obrigatória via Microsoft Artifact Signing;
- timestamp RFC3161;
- atestação de proveniência do GitHub;
- arquivo SHA-256 calculado após a assinatura.

Os binários são anexados diretamente à GitHub Release e não ficam armazenados no histórico Git.

## Rotinas

### Manutenção / conversão

Fluxo voltado a bancos sem corrupção estrutural: validação, sweep, backup e, quando selecionado, conversão/restauração para a versão escolhida.

### Recuperar banco corrompido

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
.\tools\build.ps1 -Version "1.2.32" -AllowUnsignedDev
```

O arquivo recebe `_UNSIGNED_DEV` no nome. Para distribuição pública use a Release oficial assinada.

### Tudo sai em `dist/`

`tools\build.ps1` é a única forma suportada de gerar o executável, seja chamado direto ou pela opção `1` do `FIREBIRD_TOOLBOX.bat`. Ele recria `dist/` do zero a cada execução e grava lá o aplicativo, os binários de teste, o `SHA256.txt` e o `SIGNATURE_STATUS.txt`. A raiz do repositório não recebe executável nem hash: `dist/`, `*.exe` e `*_SHA256.txt` estão no `.gitignore`, e um `.exe` solto na raiz é resíduo de um `go build` manual e deve ser apagado.

Se precisar compilar à mão para depurar, direcione a saída para `dist/`:

```powershell
$env:GOOS = "windows"
$env:GOARCH = "386"
.\tools\prepare-resources.ps1 -Version "1.2.32"
go build -trimpath -buildvcs=false -ldflags="-H=windowsgui -X main.appVersion=1.2.32" -o dist\FirebirdToolbox_v1.2.32_x86_x64.exe .
```

### Subir de versão

A versão aparece em cinco lugares e todos precisam concordar, senão a CI reprova:

1. `appVersion` em `main.go`;
2. `winres/winres.json` (`file_version`, `product_version`, `FileVersion`, `ProductVersion`);
3. o parâmetro `-Version` em `tools/build.ps1`, `tools/prepare-resources.ps1` e `tools/verify-windows-build.ps1`;
4. `.github/workflows/ci.yml`;
5. `FIREBIRD_TOOLBOX.bat`.

Acrescente a entrada no `CHANGELOG.md`, atualize `docs/LEIA-ME.md` e `docs/MANUAL.md` e rode o build. O teste `TestProjectCreditAndVersion` falha se `appVersion` ficar para trás, e `TestWindowsDistributionMetadataConfig` falha se o `winres.json` divergir.

## Publicar no GitHub

Execute `FIREBIRD_TOOLBOX.bat` e escolha a opção `2` para um fluxo guiado com confirmação antes do commit e do push. A opção `1` apenas gera o executável em `dist/` e a `3` faz as duas coisas em sequência. O passo a passo manual, com a ordem obrigatória entre CI verde, assinatura configurada e criação da tag, está em [docs/DISTRIBUICAO_WINDOWS.md](docs/DISTRIBUICAO_WINDOWS.md).

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
├── dist/                   # saída de todo build, fora do versionamento
├── docs/                   # toda a documentação
│   ├── LEIA-ME.md          # ajuda incorporada ao EXE
│   ├── MANUAL.md           # manual completo das rotinas
│   ├── DISTRIBUICAO_WINDOWS.md  # build, assinatura, publicação e Defender
│   ├── CHECKLIST_ASSINATURA.md  # checklist do Artifact Signing
│   ├── VALIDACAO_DPI.md
│   └── LEIA-ME_Restore_Firebird_Original.md
├── scripts/                # BATs originais de referência
├── winres/                 # ícone, manifesto e VersionInfo do executável
├── tools/build.ps1         # build local único x86/x64
├── tools/sign-windows.ps1  # assinatura PFX apenas para build local
├── tools/defender-preflight.ps1 # preflight Defender da Release
├── FIREBIRD_TOOLBOX.bat    # menu: gerar o .exe em dist/ e publicar
├── main.go                 # aplicação Win32
├── main_test.go            # regressões
├── go.mod
├── CHANGELOG.md
├── CONTRIBUTING.md
├── SECURITY.md
└── LICENSE
```

Documentação fica em `docs/`. A raiz guarda apenas os arquivos que o GitHub espera encontrar nela: `README.md`, `CHANGELOG.md`, `CONTRIBUTING.md`, `SECURITY.md` e `LICENSE`.

## Documentação detalhada

Consulte [docs/MANUAL.md](docs/MANUAL.md) para o histórico e os detalhes técnicos das rotinas de recuperação.

O conteúdo exibido pelo botão **Ajuda** do aplicativo é mantido em [docs/LEIA-ME.md](docs/LEIA-ME.md).

## Privacidade e segurança

Não publique bancos reais, backups, senhas SYSDBA ou relatórios contendo dados de clientes. Consulte [SECURITY.md](SECURITY.md) e [docs/DISTRIBUICAO_WINDOWS.md](docs/DISTRIBUICAO_WINDOWS.md).

## Licença

Copyright © 2026 Marcos De Mori Laiola. Todos os direitos reservados. Consulte [LICENSE](LICENSE).
