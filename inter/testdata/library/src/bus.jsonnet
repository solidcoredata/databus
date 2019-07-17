local types = import 'types.libsonnet';
local db = import 'db.libsonnet';
local ui = import 'ui.libsonnet';
{
    Types: types,
    Nodes: db + ui,
}