/*
Copyright (C) 2025 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/

import React, { useContext, useEffect, useState } from 'react';
import {
  Button,
  Typography,
  Input,
  ScrollList,
  ScrollItem,
} from '@douyinfe/semi-ui';
import { API, showError, copy, showSuccess } from '../../helpers';
import { useIsMobile } from '../../hooks/common/useIsMobile';
import { API_ENDPOINTS } from '../../constants/common.constant';
import { StatusContext } from '../../context/Status';
import { useActualTheme } from '../../context/Theme';
import { marked } from 'marked';
import { useTranslation } from 'react-i18next';
import {
  IconGithubLogo,
  IconPlay,
  IconFile,
  IconCopy,
} from '@douyinfe/semi-icons';
import { Link } from 'react-router-dom';
import NoticeModal from '../../components/layout/NoticeModal';
import {
  Moonshot,
  OpenAI,
  XAI,
  Zhipu,
  Volcengine,
  Cohere,
  Claude,
  Gemini,
  Suno,
  Minimax,
  Wenxin,
  Spark,
  Qingyan,
  DeepSeek,
  Qwen,
  Midjourney,
  Grok,
  AzureAI,
  Hunyuan,
  Xinference,
} from '@lobehub/icons';

const { Text } = Typography;

const Home = () => {
  const { t, i18n } = useTranslation();
  const [statusState] = useContext(StatusContext);
  const actualTheme = useActualTheme();
  const [homePageContentLoaded, setHomePageContentLoaded] = useState(false);
  const [homePageContent, setHomePageContent] = useState('');
  const [noticeVisible, setNoticeVisible] = useState(false);
  const isMobile = useIsMobile();
  const isDemoSiteMode = statusState?.status?.demo_site_enabled || false;
  const docsLink = statusState?.status?.docs_link || '';
  const serverAddress =
    statusState?.status?.server_address || `${window.location.origin}`;
  const endpointItems = API_ENDPOINTS.map((e) => ({ value: e }));
  const [endpointIndex, setEndpointIndex] = useState(0);
  const isChinese = i18n.language.startsWith('zh');

  const displayHomePageContent = async () => {
    setHomePageContent(localStorage.getItem('home_page_content') || '');
    const res = await API.get('/api/home_page_content');
    const { success, message, data } = res.data;
    if (success) {
      let content = data;
      if (!data.startsWith('https://')) {
        content = marked.parse(data);
      }
      setHomePageContent(content);
      localStorage.setItem('home_page_content', content);

      // 如果内容是 URL，则发送主题模式
      if (data.startsWith('https://')) {
        const iframe = document.querySelector('iframe');
        if (iframe) {
          iframe.onload = () => {
            iframe.contentWindow.postMessage({ themeMode: actualTheme }, '*');
            iframe.contentWindow.postMessage({ lang: i18n.language }, '*');
          };
        }
      }
    } else {
      showError(message);
      setHomePageContent('加载首页内容失败...');
    }
    setHomePageContentLoaded(true);
  };

  const handleCopyBaseURL = async () => {
    const ok = await copy(serverAddress);
    if (ok) {
      showSuccess(t('已复制到剪切板'));
    }
  };

  useEffect(() => {
    const checkNoticeAndShow = async () => {
      const lastCloseDate = localStorage.getItem('notice_close_date');
      const today = new Date().toDateString();
      if (lastCloseDate !== today) {
        try {
          const res = await API.get('/api/notice');
          const { success, data } = res.data;
          if (success && data && data.trim() !== '') {
            setNoticeVisible(true);
          }
        } catch (error) {
          console.error('获取公告失败:', error);
        }
      }
    };

    checkNoticeAndShow();
  }, []);

  useEffect(() => {
    displayHomePageContent().then();
  }, []);

  useEffect(() => {
    const timer = setInterval(() => {
      setEndpointIndex((prev) => (prev + 1) % endpointItems.length);
    }, 3000);
    return () => clearInterval(timer);
  }, [endpointItems.length]);

  return (
    <div className='w-full overflow-x-hidden'>
      <NoticeModal
        visible={noticeVisible}
        onClose={() => setNoticeVisible(false)}
        isMobile={isMobile}
      />
      {homePageContentLoaded && homePageContent === '' ? (
        <div className='w-full overflow-x-hidden'>
          {/* Banner 部分 */}
          <div className='w-full border-b border-semi-color-border min-h-[500px] md:min-h-[600px] lg:min-h-[700px] relative overflow-x-hidden'>
            {/* 背景模糊晕染球 */}
            <div className='blur-ball blur-ball-indigo' />
            <div className='blur-ball blur-ball-teal' />
            <div className='flex items-center justify-center h-full px-4 py-20 md:py-24 lg:py-32 mt-10'>
              {/* 居中内容区 */}
              <div className='flex flex-col items-center justify-center text-center max-w-4xl mx-auto'>
                <div className='flex flex-col items-center justify-center mb-6 md:mb-8'>
                  <h1
                    className={`text-4xl md:text-5xl lg:text-6xl xl:text-7xl font-bold text-semi-color-text-0 leading-tight ${isChinese ? 'tracking-wide md:tracking-wider' : ''}`}
                  >
                    <>
                      {t('云舒 AI')}
                      <br />
                      <span className='shine-text'>{t('高速、稳定的 AI API 中转站')}</span>
                    </>
                  </h1>
                  <p className='text-base md:text-lg lg:text-xl text-semi-color-text-1 mt-4 md:mt-6 max-w-xl'>
                    {t('更强模型 更低价格 更易落地')}
                  </p>
                  <p className='text-sm md:text-base lg:text-lg text-semi-color-text-2 mt-2 md:mt-4 max-w-xl'>
                    {t('致力于为开发者提供快速、便捷的 Web API 接口调用方案，打造稳定且易于使用的 API 接口平台，一站式集成几乎所有AI大模型。')}
                  </p>
                  
                  {/* 关键数据 */}
                  <div className='flex flex-wrap items-center justify-center gap-6 md:gap-8 mt-8 md:mt-10'>
                    <div className='text-center'>
                      <Text className='text-2xl md:text-3xl lg:text-4xl font-bold text-semi-color-text-0'>500+</Text>
                      <Text className='text-sm md:text-base text-semi-color-text-2 mt-1'>{t('大模型已经接入')}</Text>
                    </div>
                    <div className='text-center'>
                      <Text className='text-2xl md:text-3xl lg:text-4xl font-bold text-semi-color-text-0'>20万+</Text>
                      <Text className='text-sm md:text-base text-semi-color-text-2 mt-1'>{t('客户')}</Text>
                    </div>
                    <div className='text-center'>
                      <Text className='text-2xl md:text-3xl lg:text-4xl font-bold text-semi-color-text-0'>8</Text>
                      <Text className='text-sm md:text-base text-semi-color-text-2 mt-1'>{t('个地区')}</Text>
                    </div>
                    <div className='text-center'>
                      <Text className='text-2xl md:text-3xl lg:text-4xl font-bold text-semi-color-text-0'>1年+</Text>
                      <Text className='text-sm md:text-base text-semi-color-text-2 mt-1'>{t('稳定运行')}</Text>
                    </div>
                  </div>
                </div>

                {/* 操作按钮 */}
                <div className='flex flex-row gap-4 justify-center items-center mt-6 md:mt-8'>
                  <Link to='/console'>
                    <Button
                      theme='solid'
                      type='primary'
                      size={isMobile ? 'default' : 'large'}
                      className='!rounded-3xl px-8 py-2'
                      icon={<IconPlay />}
                    >
                      {t('立即开始')}
                    </Button>
                  </Link>
                  {docsLink && (
                    <Button
                      size={isMobile ? 'default' : 'large'}
                      className='flex items-center !rounded-3xl px-6 py-2'
                      icon={<IconFile />}
                      onClick={() => window.open(docsLink, '_blank')}
                    >
                      {t('文档')}
                    </Button>
                  )}
                </div>
              </div>
            </div>
          </div>
          
          {/* 优势特点部分 */}
          <div className='w-full py-16 md:py-24 lg:py-32 px-4'>
            <div className='max-w-6xl mx-auto'>
              <h2 className='text-3xl md:text-4xl lg:text-5xl font-bold text-center mb-12 md:mb-16'>
                {t('我们的优势')}
              </h2>
              
              <div className='grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-8 md:gap-10'>
                {/* 优势1 */}
                <div className='bg-semi-color-bg-1 rounded-xl p-6 md:p-8 shadow-sm'>
                  <h3 className='text-xl md:text-2xl font-bold mb-4'>{t('兼容性与支持')}</h3>
                  <p className='text-semi-color-text-2'>{t('完全兼容OpenAI接口协议，确保集成无缝。无缝对接OpenAI接口支持的应用')}</p>
                </div>
                
                {/* 优势2 */}
                <div className='bg-semi-color-bg-1 rounded-xl p-6 md:p-8 shadow-sm'>
                  <h3 className='text-xl md:text-2xl font-bold mb-4'>{t('灵活计费')}</h3>
                  <p className='text-semi-color-text-2'>{t('无需担心额度过期或封号风险，MySQL8.2超高并发不限速，智能负载均衡算法按量计费保障灵活性')}</p>
                </div>
                
                {/* 优势3 */}
                <div className='bg-semi-color-bg-1 rounded-xl p-6 md:p-8 shadow-sm'>
                  <h3 className='text-xl md:text-2xl font-bold mb-4'>{t('全球布局')}</h3>
                  <p className='text-semi-color-text-2'>{t('部署线路服务器，自动负载均衡确保快速响应。全球用户快速响应')}</p>
                </div>
                
                {/* 优势4 */}
                <div className='bg-semi-color-bg-1 rounded-xl p-6 md:p-8 shadow-sm'>
                  <h3 className='text-xl md:text-2xl font-bold mb-4'>{t('服务保障')}</h3>
                  <p className='text-semi-color-text-2'>{t('7×24小时在线支持，响应迅速，问题定位与解决。服务条款，保障计划')}</p>
                </div>
                
                {/* 优势5 */}
                <div className='bg-semi-color-bg-1 rounded-xl p-6 md:p-8 shadow-sm'>
                  <h3 className='text-xl md:text-2xl font-bold mb-4'>{t('透明计费')}</h3>
                  <p className='text-semi-color-text-2'>{t('与官方计费倍率同步，公平无猫腻，性价比最高的API源头，已有70+中转代理。')}</p>
                </div>
                
                {/* 优势6 */}
                <div className='bg-semi-color-bg-1 rounded-xl p-6 md:p-8 shadow-sm'>
                  <h3 className='text-xl md:text-2xl font-bold mb-4'>{t('Midjourney集成')}</h3>
                  <p className='text-semi-color-text-2'>{t('反代服务和中文翻译接口，实现高并发及快速响应支持最新版Midjourney Proxy Plus')}</p>
                </div>
              </div>
            </div>
          </div>
          
          {/* 服务保障部分 */}
          <div className='w-full py-16 md:py-24 lg:py-32 px-4 bg-semi-color-bg-1'>
            <div className='max-w-6xl mx-auto text-center'>
              <h2 className='text-3xl md:text-4xl lg:text-5xl font-bold mb-12 md:mb-16'>
                {t('服务保障')}
              </h2>
              
              <div className='grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-8 md:gap-10'>
                {/* 保障1 */}
                <div className='text-center'>
                  <Text className='text-2xl md:text-3xl lg:text-4xl font-bold text-semi-color-text-0 mb-4'>#1 API</Text>
                  <p className='text-semi-color-text-2'>{t('100%使用官方企业高速渠道，已稳定运行1年，承诺永久运营！')}</p>
                </div>
                
                {/* 保障2 */}
                <div className='text-center'>
                  <Text className='text-2xl md:text-3xl lg:text-4xl font-bold text-semi-color-text-0 mb-4'>8个地区</Text>
                  <p className='text-semi-color-text-2'>{t('触及8个地区，超过20万+客户')}</p>
                </div>
                
                {/* 保障3 */}
                <div className='text-center'>
                  <Text className='text-2xl md:text-3xl lg:text-4xl font-bold text-semi-color-text-0 mb-4'>24/7/365</Text>
                  <p className='text-semi-color-text-2'>{t('全天候支持我们时刻恭候您')}</p>
                </div>
                
                {/* 保障4 */}
                <div className='text-center'>
                  <Text className='text-2xl md:text-3xl lg:text-4xl font-bold text-semi-color-text-0 mb-4'>服务不间断</Text>
                  <p className='text-semi-color-text-2'>{t('便捷充值，稳定运行')}</p>
                </div>
              </div>
            </div>
          </div>
          
          {/* 框架兼容性图标 */}
          <div className='w-full py-16 md:py-24 px-4'>
            <div className='max-w-6xl mx-auto'>
              <div className='flex items-center mb-8 md:mb-12 justify-center'>
                <Text
                  type='tertiary'
                  className='text-lg md:text-xl lg:text-2xl font-light'
                >
                  {t('支持众多的大模型供应商')}
                </Text>
              </div>
              <div className='flex flex-wrap items-center justify-center gap-3 sm:gap-4 md:gap-6 lg:gap-8 max-w-5xl mx-auto px-4'>
                <div className='w-8 h-8 sm:w-10 sm:h-10 md:w-12 md:h-12 flex items-center justify-center'>
                  <Moonshot size={40} />
                </div>
                <div className='w-8 h-8 sm:w-10 sm:h-10 md:w-12 md:h-12 flex items-center justify-center'>
                  <OpenAI size={40} />
                </div>
                <div className='w-8 h-8 sm:w-10 sm:h-10 md:w-12 md:h-12 flex items-center justify-center'>
                  <XAI size={40} />
                </div>
                <div className='w-8 h-8 sm:w-10 sm:h-10 md:w-12 md:h-12 flex items-center justify-center'>
                  <Zhipu.Color size={40} />
                </div>
                <div className='w-8 h-8 sm:w-10 sm:h-10 md:w-12 md:h-12 flex items-center justify-center'>
                  <Volcengine.Color size={40} />
                </div>
                <div className='w-8 h-8 sm:w-10 sm:h-10 md:w-12 md:h-12 flex items-center justify-center'>
                  <Cohere.Color size={40} />
                </div>
                <div className='w-8 h-8 sm:w-10 sm:h-10 md:w-12 md:h-12 flex items-center justify-center'>
                  <Claude.Color size={40} />
                </div>
                <div className='w-8 h-8 sm:w-10 sm:h-10 md:w-12 md:h-12 flex items-center justify-center'>
                  <Gemini.Color size={40} />
                </div>
                <div className='w-8 h-8 sm:w-10 sm:h-10 md:w-12 md:h-12 flex items-center justify-center'>
                  <Suno size={40} />
                </div>
                <div className='w-8 h-8 sm:w-10 sm:h-10 md:w-12 md:h-12 flex items-center justify-center'>
                  <Minimax.Color size={40} />
                </div>
                <div className='w-8 h-8 sm:w-10 sm:h-10 md:w-12 md:h-12 flex items-center justify-center'>
                  <Wenxin.Color size={40} />
                </div>
                <div className='w-8 h-8 sm:w-10 sm:h-10 md:w-12 md:h-12 flex items-center justify-center'>
                  <Spark.Color size={40} />
                </div>
                <div className='w-8 h-8 sm:w-10 sm:h-10 md:w-12 md:h-12 flex items-center justify-center'>
                  <Qingyan.Color size={40} />
                </div>
                <div className='w-8 h-8 sm:w-10 sm:h-10 md:w-12 md:h-12 flex items-center justify-center'>
                  <DeepSeek.Color size={40} />
                </div>
                <div className='w-8 h-8 sm:w-10 sm:h-10 md:w-12 md:h-12 flex items-center justify-center'>
                  <Qwen.Color size={40} />
                </div>
                <div className='w-8 h-8 sm:w-10 sm:h-10 md:w-12 md:h-12 flex items-center justify-center'>
                  <Midjourney size={40} />
                </div>
                <div className='w-8 h-8 sm:w-10 sm:h-10 md:w-12 md:h-12 flex items-center justify-center'>
                  <Grok size={40} />
                </div>
                <div className='w-8 h-8 sm:w-10 sm:h-10 md:w-12 md:h-12 flex items-center justify-center'>
                  <AzureAI.Color size={40} />
                </div>
                <div className='w-8 h-8 sm:w-10 sm:h-10 md:w-12 md:h-12 flex items-center justify-center'>
                  <Hunyuan.Color size={40} />
                </div>
                <div className='w-8 h-8 sm:w-10 sm:h-10 md:w-12 md:h-12 flex items-center justify-center'>
                  <Xinference.Color size={40} />
                </div>
                <div className='w-8 h-8 sm:w-10 sm:h-10 md:w-12 md:h-12 flex items-center justify-center'>
                  <Typography.Text className='!text-lg sm:!text-xl md:!text-2xl lg:!text-3xl font-bold'>
                    30+
                  </Typography.Text>
                </div>
              </div>
            </div>
          </div>
        </div>
      ) : (
        <div className='overflow-x-hidden w-full'>
          {homePageContent.startsWith('https://') ? (
            <iframe
              src={homePageContent}
              className='w-full h-screen border-none'
            />
          ) : (
            <div className='mt-[60px]' dangerouslySetInnerHTML={{ __html: homePageContent }} />
          )}
        </div>
      )}
    </div>
  );
};

export default Home;