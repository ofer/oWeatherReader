import { ComponentFixture, TestBed } from '@angular/core/testing';
import { HistoryPageComponent } from './history-page.component';
import { ApiService, DailyAgg } from '../api.service';
import { SettingsService } from '../settings.service';
import { of } from 'rxjs';
import { MatSelectModule } from '@angular/material/select';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatProgressBarModule } from '@angular/material/progress-bar';
import { MatCardModule } from '@angular/material/card';
import { MatIconModule } from '@angular/material/icon';
import { MatTableModule } from '@angular/material/table';
import { NgxEchartsModule } from 'ngx-echarts';
import { NgIf, NgFor } from '@angular/common';
import { NoopAnimationsModule } from '@angular/platform-browser/animations';
import { HttpClientTestingModule } from '@angular/common/http/testing';

describe('HistoryPageComponent', () => {
  let component: HistoryPageComponent;
  let fixture: ComponentFixture<HistoryPageComponent>;

  const mockApiService: Partial<ApiService> = {
    getDailyAggregates: () => of([] as DailyAgg[])
  };

  const mockSettingsService: Partial<SettingsService> = {
    getIndoorDeviceModel: () => 'indoor-sensor',
    getOutdoorDeviceModel: () => 'outdoor-sensor'
  };

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      declarations: [HistoryPageComponent],
      imports: [
        HttpClientTestingModule,
        MatSelectModule,
        MatFormFieldModule,
        MatProgressBarModule,
        MatCardModule,
        MatIconModule,
        MatTableModule,
        NgxEchartsModule.forRoot({ echarts: () => import('echarts') }),
        NgIf,
        NgFor,
        NoopAnimationsModule
      ],
      providers: [
        { provide: ApiService, useValue: mockApiService },
        { provide: SettingsService, useValue: mockSettingsService }
      ]
    }).compileComponents();

    fixture = TestBed.createComponent(HistoryPageComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });

  it('should initialize with indoor and outdoor device models from settings', () => {
    expect(component.monitoringIndoor).toEqual(['indoor-sensor']);
    expect(component.monitoringOutdoor).toEqual(['outdoor-sensor']);
  });

  it('should calculate comfort rating for comfortable conditions', () => {
    const result = component.getComfortRating(72, 50);
    expect(result.rating).toBe('Comfortable');
  });

  it('should calculate comfort rating for hot conditions', () => {
    const result = component.getComfortRating(90, 50);
    expect(result.rating).toBe('Hot / Humid');
  });

  it('should calculate heat index approximately equal to temperature when below 27C', () => {
    const hi = component.calculateHeatIndex(68, 50);
    expect(hi).toBeCloseTo(68, 0);
  });

  it('should build comfort levels from aggregates', () => {
    const aggregates: DailyAgg[] = [
      {
        date: '2025-01-01', model: 'all', avgTemp: 72, highTemp: 78, lowTemp: 65,
        avgHumidity: 50, highHumidity: 70, lowHumidity: 30, modelName: 'All Selected Sensors'
      }
    ];
    const levels = component.calculateComfortLevels(aggregates);
    expect(levels.length).toBe(1);
    expect(levels[0].rating).toBe('Comfortable');
  });

  it('should sort comfort levels by date descending (newest first)', () => {
    const aggregates: DailyAgg[] = [
      { date: '2025-01-01', model: 'all', avgTemp: 65, highTemp: 70, lowTemp: 60, avgHumidity: 45, highHumidity: 60, lowHumidity: 25, modelName: 'Sensor' },
      { date: '2025-06-15', model: 'all', avgTemp: 92, highTemp: 98, lowTemp: 85, avgHumidity: 80, highHumidity: 90, lowHumidity: 70, modelName: 'Sensor' },
      { date: '2025-03-10', model: 'all', avgTemp: 80, highTemp: 85, lowTemp: 75, avgHumidity: 60, highHumidity: 70, lowHumidity: 50, modelName: 'Sensor' },
    ];
    const levels = component.calculateComfortLevels(aggregates);
    expect(levels.length).toBe(3);
    expect(levels[0].date).toBe('2025-06-15');
    expect(levels[1].date).toBe('2025-03-10');
    expect(levels[2].date).toBe('2025-01-01');
  });

  it('should preserve correct comfort rating after sorting', () => {
    const aggregates: DailyAgg[] = [
      { date: '2025-01-01', model: 'all', avgTemp: 86, highTemp: 90, lowTemp: 80, avgHumidity: 55, highHumidity: 70, lowHumidity: 40, modelName: 'Sensor' },
      { date: '2025-06-15', model: 'all', avgTemp: 72, highTemp: 78, lowTemp: 65, avgHumidity: 50, highHumidity: 70, lowHumidity: 30, modelName: 'Sensor' },
    ];
    const levels = component.calculateComfortLevels(aggregates);
    expect(levels[0].date).toBe('2025-06-15');
    expect(levels[0].rating).toBe('Comfortable');
    expect(levels[1].date).toBe('2025-01-01');
    expect(levels[1].rating).toBe('Warm / Humid');
  });

  it('should return empty array for empty aggregates', () => {
    const levels = component.calculateComfortLevels([]);
    expect(levels).toEqual([]);
  });

  it('should group comfort levels for summary', () => {
    const aggregates: DailyAgg[] = [
      { date: '2025-01-01', model: 'all', avgTemp: 72, highTemp: 78, lowTemp: 65, avgHumidity: 50, highHumidity: 70, lowHumidity: 30, modelName: 'Sensor' },
      { date: '2025-02-01', model: 'all', avgTemp: 74, highTemp: 80, lowTemp: 67, avgHumidity: 52, highHumidity: 72, lowHumidity: 32, modelName: 'Sensor' },
      { date: '2025-03-01', model: 'all', avgTemp: 65, highTemp: 70, lowTemp: 60, avgHumidity: 45, highHumidity: 60, lowHumidity: 25, modelName: 'Sensor' }
    ];
    const groups = component.getComfortGroups(aggregates);
    const comfortableGroup = groups.find(g => g.name === 'Comfortable');
    expect(comfortableGroup?.count).toBe(2);
  });

  it('should set isMonthly when daysRange > 90', () => {
    component.daysRange = 365;
    component.loadHistory();
    expect(component.isMonthly).toBeTrue();
  });

  it('should include monthly avg high and low series in temp chart', () => {
    const mockAggregates: DailyAgg[] = [
      { date: '2026-01', model: 'indoor', avgTemp: 70, highTemp: 78, lowTemp: 62, avgHighTemp: 76, avgLowTemp: 64, avgHumidity: 50, highHumidity: 70, lowHumidity: 30, modelName: 'Indoor' },
      { date: '2026-02', model: 'indoor', avgTemp: 72, highTemp: 80, lowTemp: 64, avgHighTemp: 77, avgLowTemp: 66, avgHumidity: 52, highHumidity: 72, lowHumidity: 32, modelName: 'Indoor' },
      { date: '2026-03', model: 'indoor', avgTemp: 74, highTemp: 82, lowTemp: 66, avgHighTemp: 79, avgLowTemp: 68, avgHumidity: 54, highHumidity: 74, lowHumidity: 34, modelName: 'Indoor' },
    ];

    const option = (component as any).buildTempChartFor(mockAggregates, 'Indoor');
    expect(option.series).toBeDefined();
    expect(option.series.length).toBe(5);
    // Series order in buildTempChartFor: High, Avg High, Avg, Avg Low, Low
    expect(option.series[0].name).toBe('Indoor High');
    expect(option.series[1].name).toBe('Indoor Avg High');
    expect(option.series[2].name).toBe('Indoor Avg');
    expect(option.series[3].name).toBe('Indoor Avg Low');
    expect(option.series[4].name).toBe('Indoor Low');
  });
});

describe('HistoryPageComponent loading with no outdoor sensor', () => {
  let component: HistoryPageComponent;
  let fixture: ComponentFixture<HistoryPageComponent>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      declarations: [HistoryPageComponent],
      imports: [
        HttpClientTestingModule,
        MatSelectModule,
        MatFormFieldModule,
        MatProgressBarModule,
        MatCardModule,
        MatIconModule,
        MatTableModule,
        NgxEchartsModule.forRoot({ echarts: () => import('echarts') }),
        NgIf,
        NgFor,
        NoopAnimationsModule
      ],
      providers: [
        {
          provide: ApiService,
          useValue: {
            getDailyAggregates: (models: string[]) => of([
              {
                date: '2026-01-01', model: 'indoor', avgTemp: 72, highTemp: 78, lowTemp: 65,
                avgHumidity: 50, highHumidity: 70, lowHumidity: 30, modelName: 'Indoor Sensor'
              }
            ] as DailyAgg[])
          }
        },
        {
          provide: SettingsService,
          useValue: {
            getIndoorDeviceModel: () => 'indoor-sensor',
            getOutdoorDeviceModel: () => null
          }
        }
      ]
    }).compileComponents();

    fixture = TestBed.createComponent(HistoryPageComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should have indoor but no outdoor sensor configured', () => {
    expect(component.monitoringIndoor).toEqual(['indoor-sensor']);
    expect(component.monitoringOutdoor).toEqual([]);
  });

  it('should not be stuck in loading state after init with only indoor sensor', () => {
    expect(component.loading).toBeFalse();
    expect(component.indoorAggregates.length).toBe(1);
    expect(component.hasIndoorData).toBeTrue();
  });

  it('should stop loading when loadHistory is called with only indoor sensor', (done: DoneFn) => {
    component.loading = true;
    component.loadHistory();

    setTimeout(() => {
      expect(component.loading).toBeFalse();
      expect(component.indoorAggregates.length).toBe(1);
      done();
    }, 50);
  });
});

describe('HistoryPageComponent loading with no sensors', () => {
  let component: HistoryPageComponent;
  let fixture: ComponentFixture<HistoryPageComponent>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      declarations: [HistoryPageComponent],
      imports: [
        HttpClientTestingModule,
        MatSelectModule,
        MatFormFieldModule,
        MatProgressBarModule,
        MatCardModule,
        MatIconModule,
        MatTableModule,
        NgxEchartsModule.forRoot({ echarts: () => import('echarts') }),
        NgIf,
        NgFor,
        NoopAnimationsModule
      ],
      providers: [
        {
          provide: ApiService,
          useValue: {
            getDailyAggregates: () => of([] as DailyAgg[])
          }
        },
        {
          provide: SettingsService,
          useValue: {
            getIndoorDeviceModel: () => null,
            getOutdoorDeviceModel: () => null
          }
        }
      ]
    }).compileComponents();

    fixture = TestBed.createComponent(HistoryPageComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should have no sensors configured', () => {
    expect(component.monitoringIndoor).toEqual([]);
    expect(component.monitoringOutdoor).toEqual([]);
  });

  it('should not be stuck in loading state after init with no sensors', () => {
    expect(component.loading).toBeFalse();
    expect(component.hasAnyData).toBeFalse();
  });

  it('should stop loading when loadHistory is called with no sensors', (done: DoneFn) => {
    component.loading = true;
    component.loadHistory();

    setTimeout(() => {
      expect(component.loading).toBeFalse();
      done();
    }, 50);
  });
});
