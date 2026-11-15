import test from 'node:test';
import assert from 'node:assert/strict';
import React from 'react';
import {act,create} from 'react-test-renderer';
import {MenubarTranscriptSettings} from './menubar-transcript.js';
const deferred=()=>{let resolve,reject;const promise=new Promise((a,b)=>{resolve=a;reject=b});return {promise,resolve,reject}};
async function harness(t,{preference={enabled:false,mode:'notch',displayId:'auto'},screens=[],platform='MacIntel'}={}){
 const saved=new Map(['window','localStorage','navigator'].map(key=>[key,Object.getOwnPropertyDescriptor(globalThis,key)]));
 const timers=new Map(),calls=[];let next=0,root,displayResult=screens,saveResult=null;
 const win=new EventTarget();
 win.mygo={call:async(method,...args)=>{calls.push([method,...args]);if(method==='Backend.SubtitlePreference')return preference;if(method==='Backend.SubtitleDisplays')return displayResult;if(method==='Backend.SaveSubtitleDisplay')return saveResult}};
 Object.defineProperty(globalThis,'navigator',{configurable:true,value:{platform}});
 globalThis.window=win;globalThis.localStorage={getItem:()=>null,setItem(){}};
 t.mock.method(globalThis,'setTimeout',(fn,delay)=>{assert.equal(delay,3000);timers.set(++next,fn);return next});
 t.mock.method(globalThis,'clearTimeout',id=>timers.delete(id));
 t.after(async()=>{await act(async()=>root?.unmount());for(const [key,value] of saved){if(value)Object.defineProperty(globalThis,key,value);else delete globalThis[key]}});
 await act(async()=>{root=create(React.createElement(MenubarTranscriptSettings))});
 return {calls,timers,root,select:()=>root.root.findByProps({'aria-label':'字幕屏幕'}),
  setScreens(value){displayResult=value},setSave(value){saveResult=value},
  async change(value){await act(async()=>{root.root.findByProps({'aria-label':'字幕屏幕'}).props.onChange({target:{value}})})},
  async tick(){await act(async()=>{const [id,fn]=timers.entries().next().value;timers.delete(id);fn()})}};
}
const screens=[{id:'a',name:'Display',builtIn:true,width:1512,height:982},{id:'b',name:'Display',builtIn:false,width:1920,height:1080}];
test('hydrates disconnected choice and labels duplicate screen names',async t=>{
 const f=await harness(t,{preference:{displayId:'gone',displayName:'Office'},screens});
 assert.equal(f.select().props.value,'gone');
 const options=f.select().findAllByType('option');
 assert.equal(options[1].props.disabled,true);assert.equal(options[1].children[0],'Office（未连接，暂用其他屏幕）');
 assert.equal(options[2].children[0],'Display（内置，1512×982）');
 assert.equal(options[3].children[0],'Display（外接，1920×1080）');
 f.setScreens([...screens,{id:'gone',name:'Office',builtIn:false}]);await f.tick();
 assert.equal(f.select().props.value,'gone');assert.ok(!f.select().findAllByType('option').find(o=>o.props.value==='gone').props.disabled);
});
test('user save wins over late hydration; failed save restores prior choice',async t=>{
 const pref=deferred();const f=await harness(t,{preference:pref.promise,screens});
 await f.change('b');await act(async()=>pref.resolve({displayId:'a',displayName:'Display'}));
 assert.equal(f.select().props.value,'b');assert.ok(f.calls.some(c=>c[0]==='Backend.SaveSubtitleDisplay'&&c[1]==='b'));
 f.setSave('screen disconnected');await f.change('a');assert.equal(f.select().props.value,'b');
 assert.match(f.root.root.findByProps({role:'alert'}).children.join(''),/保存失败/);
});
test('polls serially, keeps choice on errors and stops when selector unmounts',async t=>{
 const f=await harness(t,{preference:{displayId:'b'},screens});const pending=deferred();
 f.setScreens(pending.promise);await f.tick();assert.equal(f.timers.size,0);
 await act(async()=>pending.reject(new Error('offline')));assert.equal(f.timers.size,1);
 assert.equal(f.select().props.value,'b');assert.match(f.root.root.findByProps({role:'alert'}).children.join(''),/列表读取失败/);
 await act(async()=>f.root.root.findByProps({'aria-label':'字幕位置'}).props.onChange({target:{value:'menubar'}}));
 assert.equal(f.timers.size,0);assert.equal(f.root.root.findAllByProps({'aria-label':'字幕屏幕'}).length,0);
});

test('Windows offers persistent monitor selection without Mac subtitle modes',async t=>{
 const f=await harness(t,{platform:'Win32',preference:{mode:'menubar',displayId:'b'},screens:[{id:'a',name:'Panel',primary:true},{id:'b',name:'External'}]});
 assert.equal(f.root.root.findAllByProps({'aria-label':'字幕位置'}).length,0);
 assert.equal(f.select().props.value,'b');
 const labels=f.select().findAllByType('option').map(o=>o.children[0]);
 assert.deepEqual(labels,['自动选择（主显示器）','Panel（主显示器）','External（显示器）']);
 await f.change('a');
 assert.ok(f.calls.some(c=>c[0]==='Backend.SaveSubtitleDisplay'&&c[1]==='a'));
});
