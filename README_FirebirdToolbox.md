# Firebird Toolbox v1.2.20

> Para a visão geral do repositório, build e publicação no GitHub, consulte [README.md](README.md).


Utilitário portátil para Windows com um único executável x86/x64 para manutenção, recuperação e restauração de bancos Firebird **1.0, 1.5, 2.0, 2.1, 2.5, 3.0, 4.0 e 5.0**.

A v1.2.20 mantém somente três rotinas e unifica a distribuição em um único executável x86/x64, com descoberta automática da instalação/serviço/porta na máquina onde estiver rodando:

1. `Corrigir banco`
2. `Corrigir banco (avançado)`
3. `Somente restaurar`





## Conflitos iterativos e reconstrução limpa — v1.2.15

A v1.2.15 deixa de encerrar o saneamento após a primeira duplicidade. A reconstrução dos índices é testada individualmente; se surgir outra chave duplicada, o resolvedor abre novamente. Se não houver nova duplicidade e ainda assim uma relação permanecer com PK/UNIQUE inativa, o Firebird 3.0+ recebe um fallback de reconstrução limpa da relação. Depois disso, continuam obrigatórios os gates de saúde no banco restaurado:

- acesso SQL;
- sweep final;
- validação estrutural final com `gfix`;
- ausência de transações em limbo;
- zero índices de usuário inativos;
- zero `PRIMARY KEY`/`UNIQUE` inativas;
- leitura do header e indicadores transacionais com `gstat`;
- comparação de tabelas, índices e constraints com o restore de teste;
- comparação da contagem das relações diretamente afetadas pelo resgate/conflito.

O resultado detalhado fica em `RELATORIO_SAUDE_PERFORMANCE.txt`. Se um gate crítico falhar, a interface marca **NÃO APTO PARA PRODUÇÃO** e o banco não deve substituir a base em uso. Se a estrutura estiver saudável mas a recuperação tiver perda conhecida ou escolha manual entre registros, o resultado permanece **RECUPERAÇÃO PARCIAL / REVISÃO FUNCIONAL OBRIGATÓRIA**. Somente recuperação completa sem gates críticos recebe **APTO PARA PRODUÇÃO**.

## Resolvedor de conflitos de registros — v1.2.13

Quando um restore com índices inativos revela duas ocorrências diferentes para a mesma `PRIMARY KEY`/`UNIQUE`, o Toolbox não pede mais uma decisão cega por `Sim/Não`. A v1.2.13 abre uma janela própria de resolução com:

- `REGISTRO 1` e `REGISTRO 2` lado a lado;
- marcação `[DIFERENTE]` nos campos que não coincidem;
- resumo da tabela, constraint e chave problemática;
- comparação automática de BLOBs/fotos por SHA-256 no Firebird 4.0 e 5.0, quando a leitura é suportada;
- botões `Manter registro 1`, `Manter registro 2`, `Abrir comparação` e `Preservar e cancelar`;
- arquivo `COMPARACAO_CONFLITO_<TABELA>_XX.txt` preservado na pasta de recuperação;
- preservação do banco original e do banco de inspeção com todas as ocorrências;
- continuação automática do fluxo após uma escolha: remove somente as ocorrências extras na cópia de trabalho, reconstrói índices, valida, gera novo backup e repete o restore de teste.

Se o usuário escolher qual ocorrência manter quando houver conteúdo realmente diferente, o resultado permanece classificado como **RECUPERAÇÃO PARCIAL**, com a decisão registrada em `RELATORIO_RECUPERACAO.txt` e `CONFLITOS_DUPLICIDADE.txt`. Se escolher `Preservar e cancelar`, nenhuma ocorrência é removida.

## Ajustes visuais da v1.2.5

A v1.2.5 faz uma revisão de legibilidade da tela principal sem alterar o fluxo Firebird:

- amplia os rótulos de origem/destino e dos caminhos do Firebird para que os nomes apareçam completos;
- reposiciona os campos de caminho sem reduzir os botões `Procurar` / `Escolher`;
- separa `Etapa atual`, `Progresso` e `Resultado` em linhas próprias;
- amplia e reposiciona a ajuda visual de `Pular tabelas (regex)` para evitar o corte do texto `Regex opcional do gbak -skip_data (FB 3.0+)`;
- dá largura integral à descrição da etapa, evitando cortes em mensagens longas de recuperação;
- dá ao resumo final uma linha própria para exibir recuperação completa/parcial, versão, ODS e dialeto;
- aumenta a altura total da janela somente em 22 px e preserva uma área de log ampla.

## Ajuda integrada

O botão `Ajuda` agora abre uma janela própria dentro do Firebird Toolbox, sem depender do Bloco de Notas, navegador ou arquivo externo. A janela possui rolagem e reúne:

- início rápido;
- explicação das três rotinas;
- versões Firebird suportadas;
- funcionamento de `Pular tabelas (regex)` / `gbak -skip_data`;
- diferenças entre recuperação completa, parcial, backup não validado e falha;
- arquivos gerados na pasta `RECOVERY_...`;
- interpretação de erros comuns de corrupção;
- diferença entre log resumido e detalhado;
- boas práticas de segurança antes de colocar um banco recuperado em produção.

A janela também possui os botões `Copiar ajuda` e `Fechar`. O conteúdo fica incorporado ao executável e acompanha a versão instalada.

## Resgate de registros (mantido na v1.2.5)

O teste real com corrupção de página mostrou que simplesmente usar `-skip_data` em uma tabela essencial pode produzir um banco tecnicamente restaurável, mas sem os registros daquela relação. A partir da v1.2.4, esse comportamento foi substituído pelo resgate de registros, mantido na v1.2.5.

Quando o `gbak` identifica uma relação problemática, o Toolbox agora segue esta ordem:

1. Se a mensagem indicar corrupção de índice, lê os metadados da relação e tenta desativar **somente índices de usuário que não sustentam constraints**, sempre na cópia de trabalho.
2. Repete o backup sem `-skip_data`. Se funcionar, nenhum registro da relação é descartado.
3. Se ainda falhar e o Firebird for 3.0+, cria um **backup-base intermediário** usando `-skip_data` apenas para obter um banco limpo com a estrutura da tabela.
4. Restaura esse backup-base em um banco temporário de resgate.
5. Desativa temporariamente os triggers ativos da relação no banco temporário.
6. Tenta copiar todos os registros da cópia danificada usando leitura `PLAN (T NATURAL)`, evitando depender dos índices corrompidos.
7. Se a leitura natural falhar e houver uma chave primária numérica simples, divide a faixa da chave de forma progressiva até isolar os valores que realmente não podem ser lidos.
8. Reativa triggers e índices no banco temporário, gera um novo backup e executa o restore de teste obrigatório.

Se todos os registros forem copiados, o resultado volta a ser **RECUPERAÇÃO COMPLETA**. Se alguns valores de chave não puderem ser recuperados, o resultado é **RECUPERAÇÃO PARCIAL**, e o relatório lista as chaves perdidas.

O backup-base sem a relação é preservado separadamente com sufixo `_SEM_<TABELA>.BAK` para auditoria. O banco temporário de resgate também é preservado na pasta `RECOVERY_...`.

## Conflitos de chave no restore — v1.2.6

Quando o restore falha com `violation of PRIMARY or UNIQUE KEY constraint`, o Toolbox trata o caso separadamente:

1. extrai a tabela, a constraint e os valores da chave problemática do log do `gbak`;
2. não insiste automaticamente em `-one_at_a_time`, porque isso não remove a duplicidade que impede a ativação da chave;
3. preserva o restore originalmente falho;
4. refaz o restore com `gbak -inactive`, deixando índices, chaves primárias, únicas e estrangeiras inativos para que todos os registros possam ser inspecionados;
5. preserva uma cópia completa desse banco de inspeção antes de qualquer remoção;
6. conta as ocorrências da chave e compara as colunas que podem ser verificadas com segurança;
7. duplicidades logicamente idênticas podem ser reduzidas automaticamente a uma ocorrência apenas na cópia de trabalho;
8. se houver campos diferentes, todas as versões ficam preservadas no banco de inspeção e o Toolbox abre o resolvedor lado a lado; BLOBs/fotos são comparados por SHA-256 no Firebird 4.0/5.0 quando possível, e campos ainda inconclusivos permanecem claramente identificados;
9. depois do saneamento, todos os índices de usuário são reconstruídos, o banco é validado, é criado `*_SEM_DUPLICIDADES_XX.BAK` e o restore de teste completo é repetido.

Se o usuário autorizar manter uma ocorrência quando houver diferenças ou comparação inconclusiva, o resultado permanece **RECUPERAÇÃO PARCIAL**. Duplicidades estruturalmente idênticas podem concluir como recuperação completa, mas ficam registradas para auditoria.

## O que muda no modo avançado

`Corrigir banco (avançado)` agora é um fluxo de recuperação protegido para bancos com corrupção mais séria. O princípio principal é: **diagnosticar o original e executar ações destrutivas somente em uma cópia de trabalho**.

Fluxo:

1. Testa a instância/porta Firebird selecionada.
2. Executa diagnóstico profundo com `gfix -validate -full -no_update`, sem alterar o banco original.
3. Verifica transações em limbo com `gfix -list`; nenhuma transação é resolvida automaticamente.
4. Cria uma pasta `RECOVERY_<BANCO>_<DATA_HORA>` dentro da pasta de backup.
5. Cria uma cópia de trabalho do arquivo original, calcula SHA-256 durante a cópia e recalcula o SHA-256 do original ao final; aborta se tamanho, data ou conteúdo mudarem durante a cópia.
6. Valida a cópia de trabalho antes de qualquer reparo pesado.
7. Classifica a corrupção reportada em registros, índices, transações, BLOBs e páginas.
8. Se os problemas estiverem restritos a índices, evita `mend` e tenta recuperação por backup/restore.
9. Para corrupção mais séria, pede confirmação explícita antes de executar `gfix -mend -ignore`, sempre na cópia.
10. Gera um backup de recuperação. Quando necessário, usa `-ignore`; se houver transações em limbo e o backup falhar, pode oferecer nova tentativa com `-limbo`.
11. Se o `gbak` falhar em uma relação, tenta primeiro preservar os registros contornando índices corrompidos.
12. Em Firebird 3.0+, `-skip_data` pode ser usado como etapa intermediária para criar um banco limpo e depois recolocar os registros legíveis da relação.
13. Faz automaticamente um **restore de teste** do backup antes de considerá-lo válido.
14. Se o restore falhar por PRIMARY/UNIQUE duplicada, executa o fluxo controlado de conflitos descrito acima.
15. `-one_at_a_time` fica reservado para outras falhas de dados em que um banco parcial seja útil para inspeção.
16. Só depois de um restore de teste completo o fluxo segue para o destino final escolhido.
17. Gera `RELATORIO_RECUPERACAO.txt` com diagnóstico, SHA-256, ações, avisos, arquivos gerados e resultado.


## Diagnóstico técnico preservado

Quando um `gbak` de recuperação falha, o modo avançado agora continua o diagnóstico em vez de retornar apenas “backup falhou”:

- mantém `gbak -v` **somente no log técnico interno**, mesmo quando a interface está em modo resumido;
- grava `LOG_TECNICO_RECUPERACAO.txt` na mesma pasta `RECOVERY_...`, com comandos mascarados e toda a saída técnica;
- extrai as mensagens relevantes da falha e tenta identificar a última relação/tabela processada pelo `gbak`;
- nunca pula automaticamente tabelas de sistema (`RDB$`, `MON$`, `SEC$`);
- depois de `gfix -mend -ignore`, executa nova validação da cópia e registra a comparação **antes → depois**;
- captura o trecho acrescentado ao `firebird.log` durante a janela da recuperação, quando o arquivo está disponível;
- se o backup de dados falhar, tenta preservar um backup **somente de metadados** com `gbak -m`;
- em Firebird 3.0+, se uma relação de usuário puder ser identificada, pode usar `-skip_data` como etapa intermediária e tentar recuperar os registros em seguida;
- um backup final só é classificado como completo se todos os registros da relação forem reinseridos e o novo backup passar no restore de teste;
- diferencia os estados `RECUPERAÇÃO COMPLETA`, `RECUPERAÇÃO PARCIAL`, `BACKUP CRIADO MAS NÃO VALIDADO` e `NÃO RECUPERÁVEL COM FERRAMENTAS PADRÃO`.

A senha nunca é gravada no log técnico; os argumentos `-pass`/`-password` são mascarados.

## Classificação do diagnóstico

Quando o `gfix` fornece contadores, o Toolbox interpreta:

- erros de registro;
- erros de índice;
- erros de página de transação;
- erros de BLOB;
- erros de página do banco.

Erros somente de índice são tratados de forma menos agressiva porque o backup/restore pode recriar os índices. Erros de registros, BLOBs e páginas recebem classificação mais alta e exigem confirmação antes de `mend`.

## Proteção do banco original

No modo avançado:

- `gfix -mend` nunca é executado diretamente no arquivo informado como origem;
- uma cópia de trabalho é criada antes de qualquer reparo destrutivo;
- a cópia recebe SHA-256 e o original é lido novamente para confirmar que o conteúdo permaneceu estável durante a cópia;
- o restore de teste é obrigatório antes de marcar o backup como verificado;
- um restore parcial não é apresentado como sucesso completo;
- se `gfix`/`gbak` não conseguirem produzir um backup utilizável, a rotina encerra com orientação para recuperação de baixo nível, preservando original, cópia e relatório.

## Firebird antigo e códigos de retorno

Firebird 1.x e 2.0 podem reportar alguns erros na saída textual mesmo quando o processo retorna código zero. O Toolbox interpreta mensagens de falha do `gfix` em vez de confiar somente no exit code, reduzindo falsos positivos de sucesso em instalações legadas. Se um `gfix` antigo não reconhecer `-no_update`, o Toolbox não substitui esse comando por uma validação potencialmente modificadora no original: ele cria primeiro a cópia de trabalho e faz o diagnóstico estrutural completo somente nessa cópia.

## Transações em limbo

O modo avançado consulta `gfix -list`. Se houver transações em limbo:

- elas são mostradas no log e no relatório;
- o Toolbox **não decide commit ou rollback automaticamente**;
- caso o backup falhe, o usuário pode autorizar uma tentativa com `gbak -limbo`, que apenas ignora essas transações durante o backup de recuperação.

## Pular tabela corrompida — Firebird 3.0+

No modo avançado aparece o campo:

`Pular tabelas (regex)`

Exemplo:

```text
(BAD_TABLE|OLD_DATA)
```

Quando preenchido manualmente em Firebird 3.0 ou superior, o backup de recuperação adiciona `-skip_data` com essa expressão e assume que você realmente deseja excluir esses dados. Já quando o Toolbox identifica automaticamente uma tabela durante uma falha, ele tenta usar `-skip_data` apenas como **backup-base**, seguido pelo resgate de registros da relação.

## Restore de teste

O modo avançado não confia apenas no fato de o `.BAK` ter sido criado. Ele restaura o arquivo para um banco temporário usando:

- a versão de destino, se `Restaurar para` estiver selecionado; ou
- a própria versão da rotina, quando não houver destino.

Se o restore completo terminar, o Toolbox confere o header com `gstat` quando disponível. O banco temporário é removido depois do teste. Se houver tentativa parcial, o arquivo parcial é preservado para análise.

## Relatório de recuperação

Cada execução avançada cria um arquivo semelhante a:

```text
RECOVERY_DBCOM_20260810_120000_000\RELATORIO_RECUPERACAO.txt
```

O relatório registra, sem armazenar a senha:

- arquivo de origem;
- versão Firebird;
- SHA-256 e tamanho da origem copiada;
- classificação dos erros;
- número de transações em limbo detectadas;
- cópia de trabalho;
- backup de recuperação;
- restore de teste;
- destino final;
- uso de `skip_data`, quando informado ou autorizado automaticamente;
- diagnóstico antes e depois do `mend`;
- possível relação associada à falha do `gbak`;
- mensagens relevantes do `gbak`;
- backup somente de metadados, quando necessário;
- índices temporariamente desativados na cópia, quando usados para contornar corrupção de índice;
- relação submetida a resgate de registros;
- quantidade de registros recuperados;
- chaves primárias que não puderam ser recuperadas, quando houver;
- backup-base sem a relação e banco temporário de resgate;
- caminho do log técnico completo;
- trecho do `firebird.log` acrescentado durante a recuperação, quando disponível;
- ações executadas;
- avisos;
- erros;
- estado final completo/parcial/não validado/não recuperável.

## Rotina normal

`Corrigir banco` continua sendo o fluxo simples para bancos sem corrupção grave:

1. validação;
2. sweep;
3. backup;
4. restauração opcional na versão selecionada;
5. conferência do banco restaurado.

Se a validação detectar corrupção, o modo normal interrompe e orienta usar `Corrigir banco (avançado)`.

## Restauração entre versões

O Toolbox continua permitindo escolher uma versão para manutenção/backup e outra para restauração. Migrações para uma versão anterior exibem aviso adicional porque a compatibilidade regressiva não é garantida.

## Detecção do Firebird

O programa procura `gbak.exe`, `gfix.exe` e `gstat.exe` na raiz e em `bin`.

- Firebird 1.0 a 2.5: preferência por `bin`.
- Firebird 3.0 a 5.0: preferência pela raiz.

A porta pode ser detectada em `RemoteServicePort` do `firebird.conf`. O serviço Windows é procurado para informação, mas a disponibilidade operacional é confirmada pela porta TCP.

## Interface

- log integrado;
- log resumido ou detalhado;
- ícones incorporados;
- barra de progresso;
- resumo final persistente;
- confirmação de sucesso com `OK`;
- dialog de erro;
- botão Parar;
- Copiar caminho, Abrir pasta, Copiar log e Limpar log;
- Mostrar/Ocultar senha;
- campo `Pular tabelas (regex)` habilitado somente no modo avançado e em Firebird 3.0+.

## Segurança operacional

- Feche aplicações conectadas ao banco antes de manutenção ou recuperação.
- Nunca use o mesmo arquivo como origem e destino.
- Preserve o arquivo original até homologar o banco recuperado.
- `gfix -mend` pode implicar perda de dados; por isso ele só é executado na cópia e mediante confirmação.
- `-skip_data`, `-ignore`, `-limbo` e `-one_at_a_time` são recursos de resgate, não opções para uso rotineiro.
- Um banco parcialmente restaurado não é considerado recuperado por completo.


### Detecção portátil do Firebird

O Toolbox **não usa os nomes de serviço gravados nos BATs**. Os BATs estão incorporados somente como referência histórica. Em cada computador, a aplicação procura a instalação local nesta ordem:

1. `HKLM\SOFTWARE\Firebird Project\Firebird Server\Instances` nas visões 64 e 32 bits do Registry;
2. `ProgramW6432`, `Program Files` e `Program Files (x86)`;
3. `gbak.exe` disponível no `PATH`;
4. como fallback, `ImagePath` dos serviços Windows, inclusive serviços com nome personalizado.

A versão encontrada é confirmada pelo próprio `gbak -z`. A porta é lida de `RemoteServicePort` no `firebird.conf`; se não houver configuração explícita, usa o padrão oficial **3050**. O nome do serviço é apenas detectado para diagnóstico: as operações são feitas contra a instalação e a porta encontradas, sem depender de `Firebird5.0`, `Firebird3.0`, `FirebirdServerDefaultInstance` ou qualquer nome específico do computador de desenvolvimento.

## Requisitos

- Um único executável `x86_x64`: roda em Windows x86 e em Windows x64 via WOW64.
- Instalação da versão Firebird utilizada em cada etapa.
- Permissão do serviço Firebird para acessar a origem, a cópia de trabalho e a pasta de recuperação.

## Build

A distribuição oficial é gerada pelo GitHub Actions e exige Microsoft Artifact Signing. Para um build local **somente de desenvolvimento**:

```powershell
.\tools\build.ps1 -Version '1.2.20' -AllowUnsignedDev
```

Os arquivos locais sem assinatura recebem `_UNSIGNED_DEV` no nome e não devem ser distribuídos.

Para um build local com PFX confiável:

```powershell
.\tools\build.ps1 `
  -Version '1.2.20' `
  -SigningPfx 'C:\Certificados\codigo.pfx' `
  -SigningPassword 'SENHA'
```

## Assinatura

Desde a v1.2.12, a GitHub Release não possui fallback sem assinatura. O workflow exige Microsoft Artifact Signing via OIDC, timestamp RFC3161 e verificação Authenticode antes de publicar qualquer EXE. Consulte `docs/DISTRIBUICAO_WINDOWS.md`.
