// shots/registry.js —— Storyboard 可用的镜头注册表。
//
// 新增镜头 = 在这里加一行；外壳（创作台）完全不知道有哪些镜头。
import {BigNumber} from './BigNumber.jsx';
import {Compare} from './Compare.jsx';
import {Quote} from './Quote.jsx';
import {BarRace} from './BarRace.jsx';
import {Timeline} from './Timeline.jsx';
import {Choropleth} from './Choropleth.jsx';
import {ShareBreakdown} from './ShareBreakdown.jsx';
import {Flow} from './Flow.jsx';
import {SankeyFlow} from './SankeyFlow.jsx';
import {RouteGrowth} from '../compositions/RouteGrowth.jsx';

export const SHOT_REGISTRY = {
  'big-number': BigNumber,
  compare: Compare,
  quote: Quote,
  'bar-race': BarRace,
  timeline: Timeline,
  choropleth: Choropleth,
  composition: ShareBreakdown,
  flow: Flow,
  'sankey-flow': SankeyFlow,
  'route-growth': RouteGrowth,
};

export const SHOT_IDS = Object.keys(SHOT_REGISTRY);



