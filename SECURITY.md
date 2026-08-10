# Segurança

## Dados sensíveis

Nunca envie para Issues, Pull Requests ou commits:

- bancos reais de clientes (`.fdb`, `.gdb`, `.red`);
- backups (`.bak`, `.fbk`, `.nbk`);
- senhas SYSDBA;
- arquivos `.env`, certificados ou chaves;
- relatórios de recuperação contendo dados comerciais sensíveis.

O `.gitignore` bloqueia os formatos mais comuns, mas confira sempre `git status` antes de fazer commit.

## Recuperação de bancos

As rotinas avançadas podem lidar com corrupção severa. Execute-as em cópias, nunca sobre o único arquivo disponível. O Firebird Toolbox cria cópias de trabalho quando aplicável, porém backup externo e homologação do banco recuperado continuam sendo responsabilidade do operador.

## Privilégio mínimo

Desde a v1.2.12 o aplicativo não se autoeleva. O manifesto permanece `asInvoker` e não existe chamada automática a `runas`. Se uma operação específica realmente exigir privilégios administrativos, abra o programa manualmente como Administrador somente para essa operação.

## Microsoft Defender e distribuição

O Firebird Toolbox executa ferramentas administrativas legítimas do Firebird (`gfix`, `gbak`, `gstat` e `isql`) e manipula arquivos de banco. Alertas antivírus devem ser investigados; não trate automaticamente uma detecção como falso positivo.

Uma **Release oficial** deve cumprir todos estes gates:

1. VersionInfo/manifesto válidos;
2. assinatura Authenticode válida via Microsoft Artifact Signing;
3. timestamp RFC3161 presente;
4. SHA-256 calculado somente depois da assinatura;
5. atestação de proveniência do GitHub;
6. preflight do Microsoft Defender quando disponível no runner.

A Release falha se a assinatura não estiver configurada ou não puder ser validada. Builds locais sem assinatura recebem `_UNSIGNED_DEV` e não devem ser distribuídos.

## Credenciais de assinatura no GitHub

A Release usa OIDC; não armazene PFX no GitHub.

Secrets:

- `AZURE_CLIENT_ID`
- `AZURE_TENANT_ID`
- `AZURE_SUBSCRIPTION_ID`

Variables:

- `ARTIFACT_SIGNING_ENDPOINT`
- `ARTIFACT_SIGNING_ACCOUNT_NAME`
- `ARTIFACT_SIGNING_CERTIFICATE_PROFILE_NAME`

A identidade deve possuir a função **Artifact Signing Certificate Profile Signer** no recurso apropriado.

## Falso positivo

Se uma Release **assinada e verificada** ainda for detectada, não recomende exclusões permanentes. Envie o mesmo binário/hashing oficial para análise da Microsoft e preserve o resultado da submissão como evidência da Release.

## Reportar uma falha de segurança

Não publique credenciais ou bancos de clientes em Issues públicas. Compartilhe detalhes sensíveis apenas por canal privado com o responsável pelo projeto.
