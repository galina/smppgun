# Pandora SMPP gun

Реализация SMPP протокола для нагрузочного тестирования через генератор нагрузки [yandex pandora](https://yandexpandora.readthedocs.io/en/develop/).

Генератор может быть запущен как самостоятельно так и с [yandex tank](https://yandextank.readthedocs.io/en/latest/core_and_modules.html#pandora). Во втором случае есть возможность пользоваться мониторингом и другими полезными [утилитами](https://yandextank.readthedocs.io/en/latest/core_and_modules.html#handy-tools) танка.

Прежде чем запустить pandora с SMPP необходимо собрать smppgun и правильно сконфигурировать генератор.

## Сборка бенчмарка

Для сборки бенчмарка нужен golang версии не ниже 1.14.

```
$ GO=go1.14.2 make deps build
```

После успешной компиляции должен появиться файл ./smppgun

## Запуск бенчмарка

Запуск smppgun самостоятельно:

```
$ GO=go1.14.2 make run
```

Запуск в роли генератора нагрузки танка:

```
$ GO=go1.14.2 make tank
```

Также см. [Makefile](https://bb.funbox.ru/projects/A2PT/repos/smppgun/browse/Makefile) для примера запуска генератора smppgun и танка с нужными конфигурационными файлами.

## Конфигурация

### Конфигурация танка

Если бенчмарк запускается вместе с танком, то необходимо использовать соответствующий конфигурационный файл, в котором должен быть отключен [дефолтный phantom](https://github.com/yandex/yandex-tank/blob/master/yandextank/core/config/00-base.yaml) и включен pandora.
Пример конфигурации см. в [tank.example.yml](https://bb.funbox.ru/projects/A2PT/repos/smppgun/browse/tank.example.yml).

### Конфигурация pandora

Независимо от того, как запускается бенчмарк, через tank или самостоятельно, для pandora отдельно прописывается своя конфигурация.
Пример конфигурации см. в [load.example.yml](https://bb.funbox.ru/projects/A2PT/repos/smppgun/browse/load.example.yml).

### Ammo файл

Данные в ammo файле выкладываются в json формате.

Каждая отдельная запись - это либо один односоставной, либо многосоставной pdu, который при подготовке структуры ammo будет разбит на несколько сабмитов.

```
{"tag": "sm", "text": "Hello Hello Hello", "src": "src_number1", "dst": "791223456789", "enc": "latin1"}
...
```

Все нижеперечисленные поля являются обязательными при формировании ammo файла:

- `tag` - атрибут каждого запроса, строка, может быть любым.
Если в конфигурации esme клиента выставлен флаг получения отчета `deliveryReceipt`, в этом случае все входящие delivery receipt будут отдаваться в агрегатор для статистики с тегом `dlr`.
Теги могут быть полезны при построении отчетов по результатам бенчмарка;

- `text` - текст сообщения. Все сообщения передаются в поле short_message.
- `enc` - кодировка текста сообщения. Возможно указать значения latin1, ucs2, default.
- `src`, `dst` - номера отправителя и получателя.

Пример ammo файла см. в [example.ammo](https://bb.funbox.ru/projects/A2PT/repos/smppgun/browse/example.ammo).


## Результаты бенчмарка

В поле `result` конфигурации pandora указывается агрегатор, который будет собирать результаты в нужном формате. Для smppgun это всегда phout.

При запуске smppgun самостоятельно, результат будет выведен в консоль и сконфигурированный файл phout.

```
$ go1.14.2 build -v ./cmd/smppgun
./smppgun load.yml

2020-06-15T19:41:34.693+0500	INFO	cli/cli.go:210	Reading config	{"file": "load.yml"}
2020-06-15T19:41:34.697+0500	INFO	engine/engine.go:137	Pool run started	{"pool": "SMPP"}
2020-06-15T19:41:35.697+0500	INFO	cli/expvar.go:40	[ENGINE] 100 resp/s; 100 req/s; 10 users; 0 active

2020-06-15T19:41:36.697+0500	INFO	cli/expvar.go:40	[ENGINE] 102 resp/s; 102 req/s; 10 users; 0 active
...
```

### Phout

Формат phout совместим с танком, то есть танк может собирать данные из этого файла для анализа. Файл содержит агрегированные данные и подробно описан тут https://phantom-doc-ru.readthedocs.io/en/latest/analyzing_result_data.html#phout-txt

из всех полей pandora пишет в phout файл лишь часть из них:
- epoch;
- тег, строка, атрибут каждого запроса, берётся из ammo файла;
- длительность временного интервала между отправкой запроса и получением ответа(rtt) в микросекундах. Для smppgun это интервал между отправкой сабмита и получением респа на него;
- статус код протокола. Для smppgun это command_id из пакета submit_sm_resp.

## Графическое представление данных

Данные, сгенерированные в процессе работы танка, при необходимости могут быть [загружены](https://yandextank.readthedocs.io/en/latest/core_and_modules.html?highlight=overload#artifact-uploaders) для анализа в графическом виде в [overload](https://overload.yandex.net/) или в influx с последующей агрегацией данных например в grafana.
