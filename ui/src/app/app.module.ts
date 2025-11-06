import { NgModule } from '@angular/core';
import { BrowserModule } from '@angular/platform-browser';

import { AppRoutingModule } from './app-routing.module';
import { AppComponent } from './app.component';
import { HttpClientModule } from '@angular/common/http';
import { ApiService } from './api.service';
import { BrowserAnimationsModule } from '@angular/platform-browser/animations';
import { MatToolbarModule } from '@angular/material/toolbar';
import { MatButtonModule } from '@angular/material/button';
import { MatSidenavModule } from '@angular/material/sidenav';
import { MatIconModule } from '@angular/material/icon';
import { MatListModule } from '@angular/material/list';
import { LatestReportsComponent } from './latest-reports/latest-reports.component';
import { MatGridListModule } from '@angular/material/grid-list';
import { MatCardModule } from '@angular/material/card';
import { MatMenuModule } from '@angular/material/menu';
import { NavigationComponent } from './navigation/navigation.component';
import { WeatherReportDisplayComponent } from './weather-report-display/weather-report-display.component';
import { NgxEchartsModule } from 'ngx-echarts';
import { DeviceReportHistoryComponent } from './device-report-history/device-report-history.component';
import { SettingsComponent } from './settings/settings.component';
import { MatSelectModule } from '@angular/material/select';
import { MatInputModule } from '@angular/material/input';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatTableModule } from '@angular/material/table';
import { HighLowHistoryComponent } from './high-low-history/high-low-history.component';
import { HomeSummaryReportComponentComponent } from './home-summary-report-component/home-summary-report-component.component';

@NgModule({
  declarations: [
    AppComponent,
    LatestReportsComponent,
    NavigationComponent,
    WeatherReportDisplayComponent,
    DeviceReportHistoryComponent,
    SettingsComponent,
    HighLowHistoryComponent,
    HomeSummaryReportComponentComponent
  ],
  imports: [
    BrowserModule,
    AppRoutingModule,
    HttpClientModule,
    BrowserAnimationsModule,
    MatToolbarModule,
    MatButtonModule,
    MatSidenavModule,
    MatIconModule,
    MatListModule,
    MatGridListModule,
    MatCardModule,
    MatMenuModule,
    MatFormFieldModule,
    MatTableModule,
    MatInputModule,
    MatSelectModule,
    NgxEchartsModule.forRoot({
      echarts: () => import('echarts')
    })
  ],
  providers: [
    ApiService
  ],
  bootstrap: [AppComponent]
})
export class AppModule { }
