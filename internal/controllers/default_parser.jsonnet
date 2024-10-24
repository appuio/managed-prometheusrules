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
