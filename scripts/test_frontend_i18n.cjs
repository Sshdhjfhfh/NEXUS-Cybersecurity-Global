const assert = require('node:assert/strict');
const fs = require('node:fs');
const vm = require('node:vm');
const ts = require('typescript');
const catalog = require('../i18n/locales/es.json');
const source = ts.transpileModule(fs.readFileSync('web/src/lib/i18n.ts', 'utf8'), {
  compilerOptions: {module:ts.ModuleKind.CommonJS, target:ts.ScriptTarget.ES2022, esModuleInterop:true},
}).outputText;
function load({stored, fail=false}={}) {
  const exports = {};
  const document = {documentElement:{}};
  const context = {exports, document, window:{}, navigator:{language:'zh-CN'},
    localStorage:{getItem:()=>stored??null,setItem:()=>{}},
    fetch: async()=>{if(fail)throw Error('offline');return{ok:true,json:async()=>catalog}},
    require:name=>name==='react'?{}:catalog,
  };
  vm.runInNewContext(source, context);
  return {api:exports,document};
}
(async()=>{
  const {api,document}=load();
  // Labels evaluated at module load must resolve before initialization.
  assert.equal(api.t('common.save'),'Guardar');
  await api.initI18n();
  assert.equal(api.getLang(),'es');
  assert.equal(document.documentElement.lang,'es');
  const input={name:'规划',description:'unmodified',prompt:'规划',pattern:'规划'};
  const display=api.localizeApiDisplay(input,'/agents');
  assert.notEqual(display.name,input.name);
  assert.equal(display.prompt,input.prompt);
  assert.equal(display.pattern,input.pattern);
  assert.equal(input.name,'规划');
  assert.equal(api.localizeText('\n规划  '),display.name);
  assert.equal(api.localizeApiDisplay({note:'规划'},'/asset-intercept/rules').note,display.name);
  assert.equal(api.localizeApiDisplay(input,'/conversations'),input);
  assert.equal(api.localizeText('custom user data'),'custom user data');
  assert(!/\p{Script=Han}/u.test(api.localizeText('仓库 example/repo 尚未发布任何正式版本')));
  await api.setLang('ar');
  assert.equal(document.documentElement.dir,'rtl');
  assert.equal(api.localizeText('规划'),'规划');
  const offline=load({fail:true}).api;
  await offline.initI18n();
  assert.equal(offline.t('common.save'),'Guardar');
  assert.equal(offline.t('missing.key'),'[missing.key]');
  console.log('Frontend i18n checks passed: startup, offline fallback, Spanish default, RTL, display isolation');
})().catch(error=>{console.error(error);process.exitCode=1});
