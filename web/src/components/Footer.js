/*
Copyright (c) 2025 Tethys Plex

This file is part of Veloera.

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.
*/
import React, { useEffect, useState } from 'react';
import { getFooterHTML } from '../helpers';

const FooterBar = () => {
  const [footer, setFooter] = useState(getFooterHTML());
  let remainCheckTimes = 5;

  const loadFooter = () => {
    let footer_html = localStorage.getItem('footer_html');
    if (footer_html) {
      setFooter(footer_html);
    }
  };

  useEffect(() => {
    const timer = setInterval(() => {
      if (remainCheckTimes <= 0) {
        clearInterval(timer);
        return;
      }
      remainCheckTimes--;
      loadFooter();
    }, 200);
    return () => clearTimeout(timer);
  }, []);

  const PoweredByBadge = (
    <a href={`https://the.veloera.org/landing?utm_source=${window.location.hostname}&utm_campaign=footer_badage`} target='_blank' rel='noreferrer'>
      <img src='/powered_by.svg' alt='Powered by Veloera' style={{ height: '30px', verticalAlign: 'middle' }} />
    </a>
  );

  let content;
  if (footer) {
    const isMultiLine = footer.includes('<p') || footer.includes('<div') || footer.includes('<br');
    if (isMultiLine) {
      content = (
        <>
          <div className='custom-footer' dangerouslySetInnerHTML={{ __html: footer }}></div>
          <div style={{ marginTop: '5px' }}>{PoweredByBadge}</div>
        </>
      );
    } else {
      content = (
        <div style={{ display: 'inline-flex', alignItems: 'center', justifyContent: 'center', gap: '10px' }}>
          <div className='custom-footer' style={{display: 'inline-block'}} dangerouslySetInnerHTML={{ __html: footer }}></div>
          {PoweredByBadge}
        </div>
      );
    }
  } else {
    content = PoweredByBadge;
  }

  return (
    <div style={{ textAlign: 'center', paddingBottom: '5px' }}>
      {content}
    </div>
  );
};

export default FooterBar;