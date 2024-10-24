# Managed PrometheusRules

This controllers intent is to manage PrometheusRules that are otherwise immutable, eg. they are created and managed by another oprator.

## Description

This controller watches for PrometheusRules and creates new PrometheusRules by patching and filtering them through a Jsonnet parser.

By default it watches for PrometheusRules in all namespaces, except the ones that are managed by itself.
The namespaces to watch can be configured by either:
* providing a list of namespaces with the `--watch-namespace` flag and
* providing a regular expression with the `--watch-regex` flag.

By default the controller only adds the `syn: true` label to every rule, using the integrated Jsonnet parser:
```jsonnet
local input = std.extVar('rule');

local patchRules(rules) = [
    rule + { labels+: { syn: 'true' }},
    for rule in rules
];

{
    groups: [
        {
            name: group.name,
            rules: patchRules(group.rules),
        },
        for group in input.groups
    ],
}
```
This behaviour can be customized by:
* providing a custom parser with the `--external-parser` flag and
* providing a "parameters" file with the `--external-params` flag.

Like with the default parser the PrometheusRule will be passed to the parser with an external variable named `rule`.
The optional parameters file will be passed to the parser with an external variable named `params`.

If you wan't to test your custom parser use the following command:
```shell
jsonnet --ext-code <PATH_TO_TEST_RULE> --ext-code <PATH_TO_PARAMS> <YOUR_PARSER>.jsonnet
```

Only the specs of the PrometheusRule will be passed to the parser, and only the PrometheusRuleSpec will be expected as output.
