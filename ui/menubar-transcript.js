import React from 'react';
import {createSubtitleScheduler,scrollingSubtitle} from './menubar-transcript-scheduler.js';
const {createElement:h,useEffect,useRef,useState}=React;
const preference='xyz-menubar-transcript';
function readEnabled(){if(typeof window[preference]==='boolean')return window[preference];try{return localStorage.getItem(preference)==='true'}catch{return false}}
// Keep bridge state on the window: both settings and the player subscribe, but
// hydrate only once per page. Revision protects user clicks during the read.
function persistEnabled(enabled){
 const key=preference+'-writes';
 window[key]=Promise.resolve(window[key]).catch(()=>{}).then(()=>window.mygo.call('Backend.SaveSubtitlePreference',enabled));
 // A failed disk write must not break subsequent writes or the session switch.
 window[key].catch(()=>{});
 return window[key];
}
function publishEnabled(enabled){window[preference]=enabled;try{localStorage.setItem(preference,String(enabled))}catch{/* Native prefs are authoritative for opaque origins. */}window.dispatchEvent(new Event(preference))}
function toggleEnabled(){
 const next=!readEnabled();
 window[preference+'-revision']=(window[preference+'-revision']||0)+1;
 publishEnabled(next);persistEnabled(next);
}
function hydrateEnabled(){
 const key=preference+'-hydration';
 if(window[key])return;
 const revision=window[preference+'-revision']||0;
 const modeRevision=window[preference+'-mode-revision']||0;
 window[key]=Promise.resolve().then(()=>window.mygo.call('Backend.SubtitlePreference')).then(result=>{
  if((window[preference+'-mode-revision']||0)===modeRevision){
   window[preference+'-mode']=result?.mode==='menubar'?'menubar':'notch';
   window.dispatchEvent(new Event(preference));
  }
  if((window[preference+'-revision']||0)!==revision)return;
  if(typeof result?.enabled==='boolean'){publishEnabled(result.enabled);return}
  if(result?.enabled===null){
   // Only migrate a known old value, never save a default before hydration.
   let legacy;try{legacy=localStorage.getItem(preference)}catch{return}
   if(legacy==='true'||legacy==='false')persistEnabled(legacy==='true');
  }
 }).catch(()=>{delete window[key]});
}
function useEnabled(){
 const [enabled,setEnabled]=useState(readEnabled);
 useEffect(()=>{const update=()=>setEnabled(readEnabled());window.addEventListener(preference,update);hydrateEnabled();return()=>window.removeEventListener(preference,update)},[]);
 return enabled;
}
export function isWindowsSubtitlePlatform(platform=globalThis.navigator){
 return /windows|win32|win64/i.test(`${platform?.userAgent||''} ${platform?.platform||''}`);
}
function SubtitleDisplaySelector(){
 const [selection,setSelection]=useState({id:'auto',name:''});
 const [screens,setScreens]=useState([]);
 const [busy,setBusy]=useState(false),[saveError,setSaveError]=useState(''),[loadError,setLoadError]=useState(''),[preferenceError,setPreferenceError]=useState('');
 const revision=useRef(0),saving=useRef(false),alive=useRef(false);
 useEffect(()=>{
  alive.current=true;
  let active=true,timer;
  const initialRevision=revision.current;
  Promise.resolve().then(()=>window.mygo.call('Backend.SubtitlePreference')).then(result=>{
   if(active&&revision.current===initialRevision){
    setSelection({id:result?.displayId||'auto',name:result?.displayName||''});
   }
  }).catch(()=>{if(active)setPreferenceError('字幕屏幕设置读取失败，请重新打开设置')});
  const poll=async()=>{
   try{
    const result=await window.mygo.call('Backend.SubtitleDisplays');
    if(!Array.isArray(result))throw new Error('Invalid displays');
    if(active){setScreens(result);setLoadError('')}
   }catch{if(active)setLoadError('屏幕列表读取失败，稍后重试')}
   finally{if(active)timer=setTimeout(poll,3000)}
  };
  poll();
  return()=>{active=false;alive.current=false;clearTimeout(timer)};
 },[]);
 const change=async event=>{
  if(saving.current)return;
  const id=event.target.value,previous=selection;
  const next={id,name:screens.find(screen=>screen.id===id)?.name||''};
  revision.current++;saving.current=true;setBusy(true);setSaveError('');setSelection(next);
  try{
   const result=await window.mygo.call('Backend.SaveSubtitleDisplay',id);
   if(result!==null&&result!==undefined)throw new Error(String(result));
   if(alive.current)setPreferenceError('');
  }catch{
   if(alive.current){setSelection(previous);setSaveError('字幕屏幕保存失败，已保留原来的选择，请重试')}
  }finally{saving.current=false;if(alive.current)setBusy(false)}
 };
 const disconnected=selection.id!=='auto'&&!screens.some(screen=>screen.id===selection.id);
 return h('div',{className:'xyz-subtitle-screen'},
  saveError||loadError||preferenceError?h('div',{className:'xyz-setting-desc',role:'alert'},[saveError,loadError,preferenceError].filter(Boolean).join('；')):null,
  h('select',{className:'xyz-btn','aria-label':'字幕屏幕',title:'字幕显示屏',value:selection.id,disabled:busy,onChange:change},
   h('option',{value:'auto'},isWindowsSubtitlePlatform()?'自动选择（主显示器）':'自动选择（优先刘海屏）'),
   disconnected?h('option',{value:selection.id,disabled:true},`${selection.name||selection.id}（未连接，暂用其他屏幕）`):null,
   ...screens.map(screen=>{
    const duplicate=screens.filter(other=>other.name===screen.name).length>1;
    const kind=isWindowsSubtitlePlatform()?(screen.primary?'主显示器':'显示器'):(screen.builtIn?'内置':'外接');
    return h('option',{key:screen.id,value:screen.id},`${screen.name}（${kind}${duplicate?`，${screen.width}×${screen.height}`:''}）`);
   })));
}
export function MenubarTranscriptSettings(){
 const enabled=useEnabled();
 const [mode,setMode]=useState(()=>window[preference+'-mode']||'notch');
 const [busy,setBusy]=useState(false),[error,setError]=useState('');
 useEffect(()=>{const update=()=>setMode(window[preference+'-mode']||'notch');window.addEventListener(preference,update);return()=>window.removeEventListener(preference,update)},[]);
 const changeMode=async event=>{
  const next=event.target.value;setBusy(true);setError('');
  window[preference+'-mode-revision']=(window[preference+'-mode-revision']||0)+1;
  try{await window.mygo.call('Backend.SaveSubtitleMode',next);window[preference+'-mode']=next;setMode(next)}
  catch{setError('字幕位置切换失败，请重试')}
  finally{setBusy(false)}
 };
 const windows=isWindowsSubtitlePlatform();
 return h('div',{className:'xyz-subtitle-settings'},
  h('div',{className:'xyz-setting-row'},h('div',{className:'xyz-setting-label'},
   h('div',{className:'xyz-setting-name'},windows?'屏幕顶部字幕':'滚动字幕'),
   h('div',{className:'xyz-setting-desc'},'随播放进度滚动，暂停和跳转同步')),
   h('button',{type:'button',className:'xyz-btn','aria-pressed':enabled,onClick:toggleEnabled},enabled?'已开启':'已关闭')),
  h('div',{className:'xyz-subtitle-options'},
   !windows?h('select',{className:'xyz-btn','aria-label':'字幕位置',title:'字幕显示方式',value:mode,disabled:busy,onChange:changeMode},
    h('option',{value:'notch'},'屏幕顶部字幕'),h('option',{value:'menubar'},'菜单栏字幕')):null,
   windows||mode!=='menubar'?h(SubtitleDisplaySelector):null),
  error?h('div',{className:'xyz-setting-desc',role:'alert'},error):null);

}
export {scrollingSubtitle} from './menubar-transcript-scheduler.js';
export function MenubarTranscript({player,call}){
 const enabled=useEnabled(),eid=player.current?.eid;
 const current=useRef(player);current.current=player;
 const [started,setStarted]=useState(''),[result,setResult]=useState({eid:null,segments:[]});
 const last=useRef(null),startedRef=useRef('');
 const send=(text,progress=0)=>{const key=text+'|'+progress.toFixed(4);if(last.current===key)return;last.current=key;window.mygo.call('Backend.MenuSubtitle',text,progress).catch(()=>{last.current=null})};
 useEffect(()=>createSubtitleScheduler({
  audio:current.current.audioRef.current,enabled,eid,
  hasSegments:result.eid===eid&&result.segments.length>0,
  onStarted:id=>{if(startedRef.current!==id){startedRef.current=id;setStarted(id)}},
  onUpdate:audio=>{
   if(!audio||result.eid!==eid){send('');return}
   const subtitle=scrollingSubtitle(result.segments,audio.currentTime,Number.isFinite(audio.duration)?audio.duration:current.current.current?.duration);
   send(subtitle.text,subtitle.progress);
  }
 }),[enabled,eid,result]);
 useEffect(()=>{
  let alive=true;
  if(!enabled||!eid||started!==eid)return;
  call(`transcript?eid=${encodeURIComponent(eid)}`).then(d=>{
   if(alive)setResult({eid,segments:d.status==='available'&&Array.isArray(d.segments)?d.segments.filter(s=>Number.isFinite(s.startMs)&&s.startMs>=0&&typeof s.text==='string').sort((a,b)=>a.startMs-b.startMs):[]});
  }).catch(()=>{if(alive)setResult({eid,segments:[]})});
  return()=>{alive=false};
 },[enabled,eid,started,call]);
 useEffect(()=>{window.mygo.call('Backend.MenuSubtitleEnabled',enabled).catch(()=>{})},[enabled]);
 useEffect(()=>{
  const toggle=toggleEnabled;
  window.addEventListener('xyz-toggle-menubar-transcript',toggle);
  return()=>{window.removeEventListener('xyz-toggle-menubar-transcript',toggle);window.mygo.call('Backend.MenuSubtitle','',0).catch(()=>{})};
 },[]);
 return null;
}
