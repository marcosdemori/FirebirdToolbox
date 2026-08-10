# Changelog

## 1.2.14 - 2026-08-10

- Adiciona pós-recuperação obrigatório no modo avançado depois do restore final.
- Valida acesso SQL, transações em limbo, sweep final e `gfix -validate -full`.
- Bloqueia liberação quando restarem índices de usuário ou constraints `PRIMARY KEY`/`UNIQUE` inativos.
- Compara metadados e contagens das relações diretamente envolvidas com o restore de teste.
- Amplia o `gstat` para registrar OIT/OAT/OST, next transaction e sweep interval.
- Gera `RELATORIO_SAUDE_PERFORMANCE.txt` e só marca recuperação completa como `APTO PARA PRODUÇÃO` quando todos os gates críticos passam.
- Recuperações parciais estruturalmente saudáveis continuam exigindo revisão funcional e não recebem liberação automática.

## 1.2.13 - 2026-08-10

- Substitui a decisão cega de duplicidade por uma janela `Resolver conflito de registros` com `REGISTRO 1` e `REGISTRO 2` lado a lado.
- Destaca com `[DIFERENTE]` os campos escalares que não coincidem e mantém no resumo a lista completa de diferenças detectadas.
- Em Firebird 4.0/5.0, calcula SHA-256 dos BLOBs/fotos com funções nativas do Firebird e inclui os hashes na comparação; versões antigas continuam marcando BLOBs como inconclusivos em vez de assumir igualdade.
- Permite escolher explicitamente `Manter registro 1` ou `Manter registro 2`, abrir o arquivo de comparação ou `Preservar e cancelar` sem remover nenhuma ocorrência.
- Gera `COMPARACAO_CONFLITO_<TABELA>_XX.txt` na pasta de recuperação e registra a ocorrência escolhida no relatório principal e em `CONFLITOS_DUPLICIDADE.txt`.
- Após a escolha, continua automaticamente o saneamento: remove somente as ocorrências extras na cópia de trabalho, reconstrói índices, valida, gera novo backup e repete o restore de teste.
- Mantém o banco original e o snapshot de inspeção intactos; uma escolha entre registros diferentes continua classificada como recuperação parcial.
- Atualiza VersionInfo, scripts de build e documentação para 1.2.13.

## 1.2.12 - 2026-08-10

- Remove a autoelevação automática via `runas`; o aplicativo inicia como `asInvoker` e segue privilégio mínimo.
- Remove a extração dos ícones incorporados para `%TEMP%`; os arquivos ICO passam a ser interpretados em memória e convertidos em `HICON` com `CreateIconFromResourceEx`.
- GitHub Release passa a exigir Microsoft Artifact Signing via OIDC (`azure/login@v3` + `azure/artifact-signing-action@v2`); não existe mais fallback de Release sem assinatura.
- Exige timestamp RFC3161 e valida `Get-AuthenticodeSignature` antes de empacotar.
- Adiciona preflight do Microsoft Defender quando os cmdlets Defender estão disponíveis no runner.
- Gera atestações de proveniência do GitHub para os executáveis x64/x86 e o ZIP de distribuição.
- Build local sem assinatura passa a exigir `-AllowUnsignedDev`, recebe `_UNSIGNED_DEV` no nome e é marcado explicitamente como não distribuível.
- Adiciona testes de regressão para impedir retorno de autoelevação, extração de ícones para TEMP e Release oficial sem assinatura.
- Mantém integralmente a lógica de manutenção e recuperação Firebird da v1.2.11.

## 1.2.11 - 2026-08-10

- Corrige o build de Release no GitHub Actions: o PowerShell passa `-ldflags` e seu valor como argumentos separados para o `go build`, evitando o erro `invalid value "$ld" for flag -ldflags`.
- Aplica a mesma forma segura de chamada em `tools/build.ps1` e nos builds x64/x86 da CI.
- Adiciona teste de regressão para impedir o retorno da sintaxe problemática `-ldflags=$ld`.
- A Release agora valida que a versão da tag corresponde ao `appVersion` do código antes de compilar.
- Mantém a lógica de manutenção e recuperação Firebird da v1.2.10 sem alterações.

## 1.2.10 - 2026-08-10

- Corrige a falha do GitHub Actions em `TestHelpTextCoversDuplicateKeyRecovery`: a Ajuda integrada passa a citar explicitamente `gbak -inactive` no fluxo de recuperação de chaves `PRIMARY KEY`/`UNIQUE` duplicadas.
- Atualiza os workflows para `actions/checkout@v7` e `actions/setup-go@v7`, eliminando a dependência dos runtimes Node 20 usados pelas majors antigas.
- Mantém a lógica de recuperação Firebird da v1.2.9 sem alterações de regra de negócio.

## 1.2.9 - 2026-08-10

- Endurece a distribuição Windows após detecção heurística do Microsoft Defender em um build sem assinatura.
- Adiciona recursos nativos de Windows com ícone, manifesto e VersionInfo: produto, empresa/desenvolvedor, descrição, versão, copyright e nome interno.
- O manifesto passa a declarar execução como usuário atual, Windows 10 como base e Common Controls v6.
- Remove `-s -w` dos builds de distribuição para evitar binários excessivamente stripped e preservar mais metadados de diagnóstico.
- Adiciona `tools/prepare-resources.ps1`, `tools/verify-windows-build.ps1` e `tools/sign-windows.ps1`.
- CI valida os metadados dos executáveis x64/x86.
- Release aceita assinatura Authenticode opcional por certificado PFX armazenado nos secrets `WINDOWS_SIGN_CERT_BASE64` e `WINDOWS_SIGN_CERT_PASSWORD`; quando não configurado, publica explicitamente `SIGNATURE_STATUS.txt` como UNSIGNED.
- SHA-256 é calculado somente depois da etapa de assinatura.
- Nenhuma regra de manutenção/recuperação Firebird foi alterada.

## 1.2.8 - 2026-08-10

- Corrige a barra de progresso que podia permanecer visualmente incompleta ao terminar uma recuperação parcial, conflito pendente ou erro.
- Todos os estados terminais agora encerram a barra em 100%; a cor e o texto continuam distinguindo sucesso, aviso/parcial, decisão manual, cancelamento e erro.
- Adiciona atualização imediata do ProgressBar nativo do Windows antes de abrir a dialog final, evitando que a animação do controle fique atrasada em relação ao estado real da rotina.
- Atualiza `Etapa atual` no encerramento para não deixar mensagens intermediárias como `Testando o backup...` depois que o processo já terminou.
- Adiciona teste de regressão cobrindo os seis estados terminais e a atualização visual imediata da barra.
- Mantém integralmente a recuperação avançada, saneamento de duplicidades, layout e suporte Firebird 1.0 a 5.0 da v1.2.7.

## 1.2.7 - 2026-08-10

- Corrige o corte do rótulo `Regex opcional do gbak -skip_data (FB 3.0+)` na área de execução.
- Reequilibra a linha `Pular tabelas (avançado)`: mantém o campo de regex amplo e reserva 300 px para a ajuda textual.
- Adiciona teste de regressão para largura, limite direito e ausência de sobreposição entre o campo e o texto auxiliar.
- Mantém integralmente a lógica de recuperação, saneamento de duplicidades e as três rotinas da v1.2.6.

## 1.2.6 - 2026-08-10

- Trata falhas de restore por `PRIMARY KEY`/`UNIQUE KEY` duplicada como uma categoria própria, em vez de insistir automaticamente em `-one_at_a_time`.
- Extrai do `gbak` a tabela, a constraint e os valores da chave problemática, incluindo chaves compostas como `COD_DOC=31432 / COD_DOCIT=72`.
- Preserva o restore originalmente falho e refaz a restauração com `gbak -inactive`, mantendo todos os índices/constraints inativos para inspeção e saneamento controlado.
- Preserva uma cópia completa do banco restaurado com índices inativos antes de remover qualquer ocorrência duplicada.
- Compara automaticamente as ocorrências nas colunas compatíveis; BLOB/ARRAY ficam marcados como não comparáveis automaticamente.
- Duplicidades logicamente idênticas podem ser reduzidas a uma ocorrência somente na cópia de trabalho.
- Se houver campos diferentes ou comparação inconclusiva, nenhuma ocorrência é descartada silenciosamente: o Toolbox grava `CONFLITOS_DUPLICIDADE.txt`, preserva o banco com todas as versões e exige confirmação explícita para manter a primeira ocorrência física.
- Após o saneamento, reconstrói todos os índices de usuário, valida o banco, gera `*_SEM_DUPLICIDADES_XX.BAK` e repete o restore de teste completo.
- Um saneamento autorizado sobre registros diferentes/inconclusivos permanece classificado como `RECUPERAÇÃO PARCIAL`; duplicidades estruturalmente idênticas podem concluir como recuperação completa, com auditoria no relatório.
- O estado visual deixa de tratar `RECUPERAÇÃO PARCIAL` ou `AGUARDANDO DECISÃO` como erro vermelho genérico e passa a usar aviso amarelo específico.
- Amplia `RELATORIO_RECUPERACAO.txt` com conflitos de chave, ocorrências, campos diferentes, arquivos forenses, quantidade removida e backup saneado.
- Mantém as três rotinas, suporte Firebird 1.0 a 5.0, builds x64/x86, ajuda integrada, layout da v1.2.5 e os sete BATs de referência.

## 1.2.5 - 2026-08-10

- Corrige rótulos truncados na tela principal: `Banco de origem`, `Arquivo de destino`, `Pasta do backup`, `Firebird da rotina` e `Firebird da restauração`.
- Amplia as colunas de rótulos e reposiciona os campos mantendo a largura geral da interface praticamente igual.
- Reorganiza a área de execução em três linhas independentes: `Etapa atual`, barra de `Progresso` e `Resultado`.
- A mensagem de etapa passa a ter largura suficiente para textos como `Gerando backup de recuperação...`.
- O resumo final ganha uma linha própria e larga, evitando corte de estados como recuperação parcial, ODS e dialeto.
- Mantém a área de log com altura útil semelhante e aumenta a janela principal somente 22 px para acomodar a nova organização.
- Adiciona constantes e teste de regressão para impedir que as larguras mínimas de rótulos/progresso voltem a encolher.
- Preserva integralmente a recuperação de registros da v1.2.4, a ajuda integrada, as três rotinas, suporte Firebird 1.0 a 5.0, builds x64/x86, crédito no rodapé e os sete BATs de referência.

## 1.2.4 - 2026-08-10

- Corrige a estratégia de recuperação automática que antes podia preservar o banco descartando todos os dados da relação identificada pelo `gbak`.
- Antes de usar `-skip_data`, tenta contornar corrupção de índice desativando somente índices de usuário não vinculados a constraints, sempre na cópia de trabalho.
- Quando `-skip_data` ainda é necessário, passa a tratá-lo como backup-base intermediário e tenta recolocar os registros da relação em um banco limpo.
- Adiciona resgate por leitura `PLAN (T NATURAL)` para evitar dependência de índices corrompidos.
- Se a leitura completa falhar e houver chave primária numérica simples, divide a faixa da chave progressivamente para isolar somente os registros que não podem ser copiados.
- Desativa temporariamente os triggers ativos da relação somente no banco temporário de resgate e os reativa antes do backup final.
- Reconstrói índices temporariamente desativados antes da validação/uso do banco recuperado.
- O banco temporário de resgate usa a mesma versão Firebird da origem; a conversão para a versão de destino ocorre somente depois do resgate, evitando que migrações regressivas interfiram na preservação dos dados.
- Preserva separadamente o backup-base sem a relação (`*_SEM_<TABELA>.BAK`) e o banco temporário de resgate (`*_SALVAGE_<TABELA>.FDB`) para auditoria.
- Amplia o relatório com quantidade de registros recuperados, chaves não recuperadas, índices temporariamente desativados, estágio de resgate e backup-base.
- Corrige o estado visual final: recuperação parcial passa a usar aviso amarelo e dialog própria, nunca mais aparece como sucesso completo verde.
- Atualiza a confirmação do modo avançado e a Ajuda integrada para explicar o novo fluxo de preservação de registros.
- Adiciona testes de regressão para detecção de corrupção de índice, chave primária numérica, SQL de cópia NATURAL/EDS, ARRAY não suportado, contagem de registros e falhas textuais do isql.
- Mantém as três rotinas, suporte Firebird 1.0 a 5.0, builds x64/x86, crédito no rodapé e os sete BATs de referência.

## 1.2.3 - 2026-08-10

- Implementa o botão `Ajuda`, que agora abre uma janela integrada do Firebird Toolbox em vez de depender de arquivo externo/Bloco de Notas.
- Adiciona guia rolável com início rápido, três rotinas, versões suportadas, recuperação avançada, `skip_data`, estados de resultado, arquivos de recuperação, erros comuns e boas práticas.
- Adiciona ações `Copiar ajuda` e `Fechar` dentro da janela de ajuda.
- Mantém o conteúdo da ajuda incorporado ao executável e sincronizado com a versão atual.
- Adiciona teste de regressão para garantir que a ajuda cubra os fluxos e arquivos críticos de recuperação.
- Preserva integralmente a lógica de recuperação da v1.2.2, suporte Firebird 1.0 a 5.0, builds x64/x86, crédito no rodapé e os sete BATs de referência.

## 1.2.2 - 2026-08-10

- Mantém a interface em log resumido, mas preserva `gbak -v` internamente no modo avançado para diagnóstico de falhas pesadas.
- Cria `LOG_TECNICO_RECUPERACAO.txt` com saída completa e comandos com senha mascarada.
- Identifica a relação/tabela provável associada à falha do `gbak` usando o contexto técnico final.
- Extrai e grava no relatório as linhas mais relevantes da falha do backup.
- Revalida a cópia após `gfix -mend -ignore` e registra comparação antes/depois dos erros de registro, índice, transação, BLOB e página.
- Captura o trecho acrescentado ao `firebird.log` durante a recuperação, quando disponível.
- Quando o backup de dados falha, tenta preservar um backup somente de metadados com `gbak -m`.
- Em Firebird 3.0+, oferece tentativa explícita de `-skip_data` quando uma relação de usuário é identificada; tabelas de sistema nunca são excluídas automaticamente.
- Classifica backup com `-skip_data` como recuperação parcial e reflete esse estado no resumo final.
- Distingue `RECUPERAÇÃO COMPLETA`, `RECUPERAÇÃO PARCIAL`, `BACKUP CRIADO MAS NÃO VALIDADO` e `NÃO RECUPERÁVEL COM FERRAMENTAS PADRÃO`.
- Mantém restore de teste obrigatório e `-one_at_a_time` como segunda estratégia de restore.
- Preserva as três rotinas, o crédito no rodapé, suporte Firebird 1.0 a 5.0, builds x64/x86 e os sete BATs de referência.

## 1.2.1 - 2026-08-10

- Adiciona ao rodapé da interface a identificação `Projeto feito por Marcos De Mori Laiola`.
- Mantém o status operacional à esquerda e a assinatura do projeto à direita, sem alterar o fluxo das rotinas.
- Preserva integralmente a recuperação avançada, suporte Firebird 1.0 a 5.0, builds x64/x86 e os sete BATs de referência.

## 1.2.0 - 2026-08-10

- Transforma `Corrigir banco (avançado)` em um fluxo protegido de recuperação de corrupção.
- Executa diagnóstico profundo do original com `gfix -validate -full -no_update`, sem alteração do banco nesta etapa.
- Classifica contadores de erros de registro, índice, transação, BLOB e páginas quando reportados pelo gfix.
- Detecta transações em limbo com `gfix -list` e nunca decide commit/rollback automaticamente.
- Cria pasta de recuperação isolada, cópia de trabalho e SHA-256 antes de reparos destrutivos.
- Detecta alteração de tamanho/data do banco durante a cópia e também recalcula o SHA-256 do original ao final, cancelando se o conteúdo tiver mudado.
- Confere a cópia de trabalho em modo somente leitura antes de executar qualquer mend.
- Evita `gfix -mend` quando os erros estão restritos a índices.
- Para corrupção mais séria, exige confirmação e executa `gfix -mend -ignore` somente na cópia de trabalho.
- Adiciona backup de recuperação com `-ignore` quando necessário e tentativa opcional com `-limbo` quando transações em limbo bloqueiam o backup.
- Adiciona campo `Pular tabelas (regex)` e suporte a `gbak -skip_data` em Firebird 3.0+.
- Faz restore de teste obrigatório do backup antes de considerá-lo verificado.
- Adiciona tentativa opcional `-one_at_a_time` quando o restore de teste normal falha; resultados parciais são preservados mas nunca apresentados como sucesso completo.
- Gera `RELATORIO_RECUPERACAO.txt` com SHA-256, diagnóstico, ações, avisos, arquivos gerados e resultado.
- Melhora a detecção de falhas textuais do gfix para instalações antigas que podem retornar exit code zero mesmo em erro.
- Se um gfix legado não reconhecer `-no_update`, evita executar validação potencialmente modificadora no original e desloca o diagnóstico completo para a cópia de trabalho.
- O modo normal agora interrompe ao detectar sinais de corrupção e orienta usar o modo avançado.
- Mantém as três rotinas, suporte Firebird 1.0 a 5.0, builds x64/x86, log integrado e interface da linha v1.1.x.

## 1.1.7 - 2026-08-10

- Corrige falha de inicialização introduzida na v1.1.6 após a remoção da quarta rotina.
- O combo de rotinas agora é montado dinamicamente a partir da lista real de rotinas, evitando acesso fora do limite.
- Mantém somente `Corrigir banco`, `Corrigir banco (avançado)` e `Somente restaurar`.
- Adiciona proteção global de inicialização: um panic inesperado passa a exibir uma dialog de erro em vez de fechar silenciosamente.
- Adiciona teste para garantir que os nomes exibidos no combo permaneçam sincronizados com as definições de rotina.

## 1.1.6 - 2026-08-10

- Remove a rotina `Converter backup 3.0 -> 2.5` da interface.
- Mantém somente três rotinas: `Corrigir banco`, `Corrigir banco (avançado)` e `Somente restaurar`.
- Remove o fluxo dedicado de conversão do código para evitar opção oculta ou comportamento legado não utilizado.
- Remove `scripts/converter_fb30_para_fb25.bat` do pacote e do conteúdo incorporado ao executável.
- Mantém suporte às versões Firebird 1.0, 1.5, 2.0, 2.1, 2.5, 3.0, 4.0 e 5.0.
- Mantém a restauração entre versões pelas rotinas normais, inclusive com aviso de compatibilidade ao selecionar uma versão de destino anterior.
- Preserva interface, ícones, progresso, log integrado, detecção automática e confirmação final da v1.1.5.

## 1.1.5 - 2026-08-10

- Expande as versões disponíveis para Firebird 1.0, 1.5, 2.0, 2.1, 2.5, 3.0, 4.0 e 5.0.
- Adiciona build Windows x86/32-bit para ambientes legados, mantendo x64 como build principal.
- Adiciona defaults e detecção de instalação para todas as linhas suportadas em Program Files e Program Files (x86).
- Torna a busca por `gbak.exe`, `gfix.exe` e `gstat.exe` tolerante aos layouts em raiz e em `bin`.
- Trata Firebird 1.0 a 2.5 como layouts legados que preferem `bin`, mantendo 3.0 a 5.0 com preferência pela raiz.
- Libera a seleção da versão de origem também no modo `Corrigir banco (avançado)`.
- Adiciona sufixos automáticos de destino 10, 15, 20, 21, 25, 30, 40 e 50.
- Adiciona confirmação adicional antes de migrações para uma versão anterior do Firebird.
- Passa a usar Service Manager nas restaurações 1.5 a 5.0 e modo local no Firebird 1.0 por compatibilidade com instalações antigas.
- Corrige o uso de `-REP`: quando o destino existe, `-rep` substitui `-c` em vez de combinar os dois switches.
- Mantém a rotina especial `Converter backup 3.0 -> 2.5` por compatibilidade com o fluxo original.
- Preserva interface, ícones, progresso, log resumido, confirmação final e botão Mostrar/Ocultar senha da v1.1.4.

## 1.1.4 - 2026-08-10

- Corrige definitivamente o botão `Mostrar senha`/`Ocultar senha`, alternando o estilo `ES_PASSWORD` e forçando o redesenho do campo sem alterar o conteúdo.
- Adiciona teste de regressão para a alternância do estilo de senha.
- Refaz o cabeçalho da janela com ícone do Firebird Toolbox, título e descrição mais bem distribuídos.
- Substitui os títulos embutidos dos group boxes por cabeçalhos de seção independentes com ícones visíveis.
- Passa a desenhar os principais botões diretamente, garantindo ícone + texto mesmo sem Common Controls v6/manifesto.
- Destaca `Executar` como ação primária e `Parar` como ação de interrupção.
- Move `Detalhado`, `Limpar` e `Copiar` para o cabeçalho da área de log, reduzindo poluição na área de execução.
- Ajusta espaçamentos, larguras e alinhamentos das áreas de origem/destino, Firebird, progresso e log.
- Mantém integralmente as rotinas Firebird e a confirmação final com botão `OK` da v1.1.3.

## 1.1.3 - 2026-08-10

- Adiciona ícones visuais incorporados ao executável para as seções e os principais botões.
- Adiciona ícone próprio do Firebird Toolbox na janela.
- Adiciona barra de progresso nativa com estado normal, interrompido e erro.
- Mantém um resumo compacto do resultado visível após a conclusão.
- Exibe uma dialog de sucesso com botão `OK` e os principais dados da execução.
- Exibe uma dialog de erro com a causa resumida e orientação para consultar o log.
- Adiciona ação `Copiar caminho` para o banco restaurado ou backup final.
- Mantém `Abrir pasta`, `Copiar log` e `Limpar log` no fluxo principal.
- Melhora a acentuação dos textos próprios da interface e das mensagens de resultado.
- Preserva as rotinas de gfix/gbak/gstat, a detecção automática de serviço/porta e o log resumido da v1.1.2.

## 1.1.2 - 2026-08-10

- Remove a dependencia de nomes fixos de servico como `Firebird5.0` para considerar o servidor disponivel.
- Deixa o campo de servico em modo `Automatico` por padrao.
- Detecta `RemoteServicePort` diretamente no `firebird.conf` da instalacao selecionada.
- Testa a porta TCP do Firebird antes de iniciar operacoes de rede.
- Quando a porta digitada nao responde e a porta configurada responde, usa automaticamente a porta detectada.
- Procura o servico Windows Firebird em segundo plano e mostra o nome encontrado sem gerar falso erro 1060.
- Decodifica a saida OEM do Windows antes de envia-la ao log, corrigindo textos como `servico` que apareciam com caractere quebrado.
- No modo resumido, deixa de exibir comandos completos e saidas tecnicas de sucesso.
- Resume o `gstat` em ODS, dialeto e tamanho de pagina.
- Mantem o header completo do `gstat` no modo detalhado.
- Adiciona marcadores claros de sucesso para validacao, sweep, backup, restauracao e dialeto.
- Confirma que o arquivo de destino realmente foi criado apos o `gbak`.

## 1.1.1 - 2026-08-10

- Adiciona log resumido por padrao.
- Remove `-v` do `gbak` no modo resumido para evitar milhares de linhas de privilegios e indices.
- Mantem modo detalhado opcional para diagnostico.

## 1.1.0 - 2026-08-10

- Redesenha a interface em formato mais compacto e organizado por secoes.
- Integra o log de execucao na propria janela.
- Remove a necessidade de abrir console externo para gfix/gbak/gstat.
- Separa `Rotina`, `Versao da rotina` e `Restaurar para`.
- Permite combinar correcao em uma versao com restauracao em outra.
- Adiciona pasta de backup selecionavel.
- Adiciona botao Parar com cancelamento do processo Firebird atual.
- Adiciona Limpar log e Copiar log.
- Mascara a senha do Firebird nas linhas de comando exibidas.
- Leva as perguntas do modo avancado 3.0 para dialogs da propria aplicacao.
- Preserva a rotina de conversao 3.0 -> 2.5 dos BATs originais.
- Mantem ajuste de dialeto para 1 apos restauracao no Firebird 5.0.
- Evita `taskkill` global dos processos Firebird.
